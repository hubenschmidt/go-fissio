# go-fissio

A Go framework for building agentic LLM pipelines with RAG support.

![Research Assistant](spec/screenshots/research_assistant.png)

## Features

- **Visual Pipeline Editor** — Drag-and-drop node configuration
- **Multi-provider LLMs** — OpenAI, Anthropic, Ollama
- **Embedding Support** — Generate embeddings for semantic search
- **Vector Store** — In-memory or PostgreSQL (pgvector)
- **RAG Tools** — `similarity_search`, `index_document`
- **SSE Streaming** — Token-by-token response streaming
- **Pipeline Templates** — Pre-built RAG and code generation patterns

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         GO-FISSIO                               │
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Pipeline  │    │   Engine    │    │   Server    │         │
│  │   Config    │───▶│  Executor   │◀──▶│   API       │         │
│  └─────────────┘    └──────┬──────┘    └─────────────┘         │
│                            │                                    │
│         ┌──────────────────┼──────────────────┐                │
│         ▼                  ▼                  ▼                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │    LLM      │    │   Tools     │    │   Vector    │         │
│  │   Client    │    │  Registry   │    │   Store     │         │
│  │ +Embedding  │    └──────┬──────┘    └──────┬──────┘         │
│  └─────────────┘           │                  │                │
│                            ▼                  ▼                │
│                     ┌─────────────────────────────┐            │
│                     │   Semantic Search Tools     │            │
│                     │  • similarity_search        │            │
│                     │  • index_document           │            │
│                     └─────────────────────────────┘            │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

```bash
cp .env.example .env
# Edit .env with your API keys

docker compose up
```

- Editor: http://localhost:3001
- Server: http://localhost:8000

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OPENAI_API_KEY` | OpenAI API key |
| `ANTHROPIC_API_KEY` | Anthropic API key |
| `OLLAMA_URL` | Ollama server URL (default: http://localhost:11434) |
| `DATABASE_URL` | PostgreSQL DSN for pgvector (optional) |
| `FISSIO_DATA_DIR` | Data directory for SQLite (default: ./data) |

## Package Structure

| Package | Description |
|---------|-------------|
| `config` | Pipeline schema, node/edge types |
| `core` | Error types, messages, model config |
| `engine` | DAG execution engine |
| `llm` | LLM provider clients + embeddings |
| `tools` | Tool registry and built-in tools |
| `vector` | Vector store interface + implementations |
| `pipelines` | Pre-built pipeline templates |
| `server` | HTTP server with SSE |

## LLM Providers

| Provider | Chat | Embeddings | Default Model |
|----------|------|------------|---------------|
| OpenAI | Yes | Yes | `text-embedding-3-small` |
| Anthropic | Yes | No | — |
| Ollama | Yes | Yes | `nomic-embed-text` |

## Node Types

| Type | Description | Tools |
|------|-------------|-------|
| `llm` | Simple LLM call | No |
| `worker` | Agentic tool loop | Yes |
| `router` | Classifies and routes | No |
| `gate` | Validates input | No |
| `aggregator` | Combines outputs | No |
| `orchestrator` | Dynamic decomposition | No |
| `evaluator` | Quality scoring | No |
| `synthesizer` | Synthesizes inputs | No |
| `coordinator` | Distributes work | No |

## Built-in Tools

| Tool | Description |
|------|-------------|
| `fetch_url` | Fetches content from a URL |
| `web_search` | Web search via Tavily API |
| `similarity_search` | Semantic search over vector store |
| `index_document` | Index documents into vector store |

## RAG (Retrieval-Augmented Generation)

RAG augments LLM responses with context retrieved from your own documents.

### How it works

1. **Index** — Documents are embedded and stored in a vector store
2. **Retrieve** — User queries are embedded and matched against stored vectors
3. **Generate** — Retrieved context is injected into the LLM prompt

### Vector Store Options

**In-memory** (default, no dependencies):
```go
store := vector.NewMemoryStore()
```

**PostgreSQL with pgvector** (production):
```go
store, err := vector.NewPgVector(ctx, "postgres://user:pass@localhost/db")
```

### Embedding Models

| Provider | Model | Dimensions |
|----------|-------|------------|
| OpenAI | `text-embedding-3-small` | 1536 |
| OpenAI | `text-embedding-3-large` | 3072 |
| Ollama | `nomic-embed-text` | 768 |

### Usage Example

```go
client := fissio.NewUnifiedClient(fissio.UnifiedConfig{
    OpenAIKey: os.Getenv("OPENAI_API_KEY"),
})

store := vector.NewMemoryStore()

// Index documents
docs := []vector.Document{
    {ID: "1", Content: "Users table contains customer info..."},
    {ID: "2", Content: "Orders table tracks purchases..."},
}

// Embed and store
for i, doc := range docs {
    resp, _ := client.Embed(ctx, "text-embedding-3-small", doc.Content)
    docs[i].Embedding = resp.Embedding
}
store.Upsert(ctx, docs)

// Search
queryEmbed, _ := client.Embed(ctx, "text-embedding-3-small", "customer data")
results, _ := store.Search(ctx, queryEmbed.Embedding, 5)
```

### Pipeline Templates

**RAG Pipeline:**
```go
pipeline := pipelines.NewRAGPipeline("You are a helpful assistant.")
```

**Code Generation Pipeline:**
```go
pipeline := pipelines.NewCodeGenPipeline("SQL")
```

## Development

### Prerequisites

- Go 1.24+
- Node.js 18+ (for client)
- Docker (optional)

### Run locally

```bash
# Server
go run ./cmd/server

# Client
cd client && npm install && npm run dev
```

### Build

```bash
go build -o fissio ./cmd/server
```

## License

MIT
