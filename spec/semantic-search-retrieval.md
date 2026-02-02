# Semantic Search & Retrieval Infrastructure

## Goal

Extend go-fissio to support semantic search, embedding-based retrieval, and code generation pipelines. This positions the framework for data intelligence applications: catalog search, context-aware code generation, and RAG patterns.

---

## Why This Matters

LLMs are effective at generating code, but they lack **context**. When a user asks "write SQL to calculate monthly churn," the LLM doesn't know:
- What tables exist in their data warehouse
- What columns are available
- How tables relate to each other
- What naming conventions the organization uses

The solution is **retrieval-augmented generation (RAG)**: before generating code, retrieve relevant context (schemas, documentation, examples) and include it in the prompt. This transforms a generic LLM into a context-aware assistant that understands the user's specific environment.

**When to use vector search vs. context stuffing:**

| Approach | Use When |
|----------|----------|
| Context stuffing | Small corpus (~1000s items), fits in context window |
| Vector search | Large corpus (~100K+ items), need semantic similarity |

Vector search is a **smart filter** that reduces context size by selecting only the most relevant items before the LLM sees them.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        GO-FISSIO                                │
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Pipeline  │    │   Engine    │    │   Server    │         │
│  │   Config    │───▶│  Executor   │◀──▶│   API       │         │
│  └─────────────┘    └──────┬──────┘    └─────────────┘         │
│                            │                                    │
│         ┌──────────────────┼──────────────────┐                │
│         ▼                  ▼                  ▼                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │    LLM      │    │   Tools     │    │   Vector    │  NEW    │
│  │   Client    │    │  Registry   │    │   Store     │◀────    │
│  │ +Embedding  │◀   └──────┬──────┘    └──────┬──────┘         │
│  └─────────────┘           │                  │                │
│         NEW ▲              │                  │                │
│             │              ▼                  ▼                │
│             │       ┌─────────────────────────────┐            │
│             └───────│   Semantic Search Tools     │  NEW       │
│                     │  • similarity_search        │◀────       │
│                     │  • index_document           │            │
│                     └─────────────────────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Embedding Infrastructure

### Problem

LLMs operate on text, but semantic search requires **vector representations**. When a user asks "find tables related to customer revenue," we need to:
1. Convert the query into a vector (embedding)
2. Compare it against pre-computed vectors of table descriptions
3. Return the most similar matches

Currently, go-fissio's LLM client only supports chat completions. There's no way to generate embeddings, which means we can't build semantic search without calling external APIs directly—breaking the clean abstraction the framework provides.

### Solution

Extend the LLM client layer with an `EmbeddingClient` interface. This keeps embedding generation within the same unified client pattern, with automatic provider routing.

### Interface

```go
// llm/types.go
type EmbeddingResponse struct {
    Embedding  []float64 `json:"embedding"`
    TokenCount int       `json:"token_count"`
}

// llm/client.go
type EmbeddingClient interface {
    Embed(ctx context.Context, model, input string) (*EmbeddingResponse, error)
    EmbedBatch(ctx context.Context, model string, inputs []string) ([]EmbeddingResponse, error)
}
```

### Provider Support

| Provider | Endpoint | Batch Support | Default Model |
|----------|----------|---------------|---------------|
| OpenAI | `/v1/embeddings` | Yes | `text-embedding-3-small` |
| Ollama | `/api/embed` | Yes | `nomic-embed-text` |
| Anthropic | N/A | No | Skip (use OpenAI/Ollama) |

### Files

| Action | File |
|--------|------|
| Modify | `llm/types.go` - add EmbeddingResponse |
| Modify | `llm/client.go` - add EmbeddingClient interface |
| Modify | `llm/openai.go` - implement Embed/EmbedBatch |
| Create | `llm/ollama.go` - implement Embed |
| Modify | `llm/unified.go` - implement EmbeddingClient, route by model |

---

## Phase 2: Vector Store

### Problem

Embeddings are arrays of floats (typically 1536 dimensions). To make them useful, we need:
1. **Persistent storage** - Recomputing embeddings is expensive
2. **Efficient similarity search** - Brute-force doesn't scale beyond ~10K documents
3. **Metadata association** - Each embedding needs context

### Solution

Create a `VectorStore` interface with multiple implementations:
- **In-memory** for development (simple, no dependencies)
- **pgvector** for production (leverages existing PostgreSQL)

### Interface

```go
// vector/store.go
type Document struct {
    ID        string            `json:"id"`
    Content   string            `json:"content"`
    Embedding []float64         `json:"embedding,omitempty"`
    Metadata  map[string]any    `json:"metadata,omitempty"`
}

type SearchResult struct {
    Document Document  `json:"document"`
    Score    float64   `json:"score"`  // cosine similarity
}

type VectorStore interface {
    Upsert(ctx context.Context, docs []Document) error
    Search(ctx context.Context, embedding []float64, topK int) ([]SearchResult, error)
    Delete(ctx context.Context, ids []string) error
    Close() error
}
```

### Schema (pgvector)

```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    content TEXT NOT NULL,
    embedding vector(1536),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_documents_embedding ON documents
    USING hnsw (embedding vector_cosine_ops);
```

### Files

| Action | File |
|--------|------|
| Create | `vector/store.go` - interface and types |
| Create | `vector/cosine.go` - similarity math |
| Create | `vector/memory.go` - in-memory implementation |
| Create | `vector/pgvector.go` - PostgreSQL implementation |

---

## Phase 3: Semantic Search Tools

### Problem

Embeddings and vector stores are infrastructure—not directly usable by LLM agents. Agents interact through **tools**. A Worker node needs a tool that:
1. Accepts natural language queries
2. Embeds the query
3. Searches the vector store
4. Returns formatted results

### Solution

Create tools that wrap embedding + vector store operations:

- **`similarity_search`** - Core retrieval tool for RAG
- **`index_document`** - Dynamic document ingestion

### Implementation

```go
// tools/similarity_search.go
type SimilaritySearchTool struct {
    store    vector.VectorStore
    embedder llm.EmbeddingClient
    model    string
}

func (t *SimilaritySearchTool) Name() string { return "similarity_search" }

func (t *SimilaritySearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
    var req struct {
        Query string `json:"query"`
        TopK  int    `json:"top_k"`
    }
    json.Unmarshal(args, &req)

    if req.TopK == 0 {
        req.TopK = 5
    }

    // Embed query
    resp, err := t.embedder.Embed(ctx, t.model, req.Query)
    if err != nil {
        return "", err
    }

    // Search
    results, err := t.store.Search(ctx, resp.Embedding, req.TopK)
    if err != nil {
        return "", err
    }

    return formatSearchResults(results), nil
}
```

### Files

| Action | File |
|--------|------|
| Create | `tools/similarity_search.go` |
| Create | `tools/index_document.go` |

---

## Phase 4: Configuration & Integration

### Problem

Users face setup burden: instantiate embedder, create vector store, wire tools, register with registry. The framework should "just work" for the common case.

### Solution

Extend server config to handle vector infrastructure automatically:
- PostgreSQL DSN → use pgvector
- Otherwise → use in-memory
- Auto-register semantic search tools

### Config Extension

```go
type Config struct {
    Client       llm.Client
    Registry     *tools.Registry
    OllamaURL    string
    DatabaseDSN  string

    // NEW
    VectorStore  vector.VectorStore  // Optional: inject custom store
    EmbedModel   string              // Default embedding model
}
```

### Files

| Action | File |
|--------|------|
| Modify | `server/server.go` - config extension |
| Modify | `fissio.go` - re-exports |

---

## Phase 5: Pipeline Templates

### Problem

Building RAG pipelines requires understanding node types, edges, prompts. Users want to solve problems, not learn framework internals.

### Solution

Pre-built pipeline templates:

```go
// pipelines/rag.go
func NewRAGPipeline(systemPrompt string) *config.PipelineConfig {
    return config.NewPipeline("rag", "Retrieval-Augmented Generation").
        Node("retriever", config.NodeWorker).
            Prompt("Search for relevant context to answer the user's question.").
            Tools("similarity_search").
            MaxIter(3).
            Done().
        Node("generator", config.NodeLLM).
            Prompt(systemPrompt).
            Done().
        Edge("retriever", "generator").
        Build()
}

// pipelines/codegen.go
func NewCodeGenPipeline(language string) *config.PipelineConfig {
    return config.NewPipeline("codegen", "Code Generator").
        Node("context", config.NodeWorker).
            Prompt("Find relevant schemas, documentation, and examples.").
            Tools("similarity_search").
            Done().
        Node("generator", config.NodeLLM).
            Prompt(fmt.Sprintf("Generate %s code using the provided context.", language)).
            Done().
        Node("validator", config.NodeEvaluator).
            Prompt("Check the generated code for syntax errors and issues.").
            Done().
        Edge("context", "generator").
        Edge("generator", "validator").
        Build()
}
```

### Files

| Action | File |
|--------|------|
| Create | `pipelines/rag.go` |
| Create | `pipelines/codegen.go` |

---

## Dependencies

```
github.com/pgvector/pgvector-go  # pgvector Go client
```

---

## Usage Example

```go
client := fissio.NewUnifiedClient(fissio.UnifiedConfig{
    OpenAIKey: os.Getenv("OPENAI_API_KEY"),
})

vectorStore, _ := vector.NewPgVector(os.Getenv("DATABASE_URL"))

// Index documents
docs := []vector.Document{
    {ID: "1", Content: "The users table contains customer information..."},
    {ID: "2", Content: "The orders table tracks all purchase transactions..."},
}
vectorStore.Upsert(ctx, docs)

// Create and run RAG pipeline
pipeline := pipelines.NewRAGPipeline("You are a helpful data assistant.")
engine := fissio.NewEngine(pipeline, fissio.EngineConfig{
    Client:      client,
    VectorStore: vectorStore,
})

result, _ := engine.Run(ctx, "What tables store customer data?")
```

---

## Verification

1. Unit test: `vector/memory_test.go` - in-memory store operations
2. Integration test: Embed → store → search → verify results
3. End-to-end: RAG pipeline with sample documents
4. API test: `/chat` with pipeline using `similarity_search` tool
