# go-fissio

A Go framework for building agentic LLM pipelines with RAG support.

![Research Assistant](spec/screenshots/research_assistant.png)
![Compose Mode](spec/screenshots/compose.png)

## Features

- **Visual Pipeline Editor** — Drag-and-drop node configuration
- **Multi-provider LLMs** — OpenAI, Anthropic, Ollama
- **Embedding Support** — Generate embeddings for semantic search
- **Vector Store** — In-memory or PostgreSQL (pgvector)
- **RAG Tools** — `similarity_search`, `index_document`
- **SSE Streaming** — Token-by-token response streaming
- **Pipeline Templates** — Pre-built RAG and code generation patterns

## Usage Example

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

## Frontend Editor

go-fissio ships with a visual pipeline editor built in SolidJS (`client/`). It supports drag-and-drop node configuration, edge wiring, and template loading.

### Development Mode (hot-reload)

Run the Go server and SolidJS client separately via Docker Compose:

```bash
cp .env.example .env
# Edit .env with your API keys

docker compose up
```

- Editor: http://localhost:3001 (Vite dev server with HMR)
- API: http://localhost:8000

The `DEV=1` environment variable tells the Go server to serve only API routes. The SolidJS client runs on its own container and proxies API calls to the server.

### Production Mode (embedded)

The editor is embedded directly into the Go binary using `go:embed`. No separate Node.js process is needed.

```go
package main

import (
    "log"
    "net/http"
    "os"

    "github.com/hubenschmidt/go-fissio"
)

func main() {
    client := fissio.NewUnifiedClient(fissio.UnifiedConfig{
        OpenAIKey: os.Getenv("OPENAI_API_KEY"),
    })

    srv, _ := fissio.NewServer(fissio.ServerConfig{Client: client})
    defer srv.Close()

    mux := http.NewServeMux()
    mux.Handle("/", fissio.EditorHandler())              // embedded SPA
    mux.Handle("/api/", http.StripPrefix("/api", srv.Handler()))

    log.Fatal(http.ListenAndServe(":8000", mux))
}
```

Build and run as a single binary:

```bash
go build -o fissio-server ./cmd/server
./fissio-server
# Editor + API both on http://localhost:8000
```

### Rebuilding the Embedded Editor

When making changes to the SolidJS client, rebuild the embedded assets:

```bash
cd client && npm run build
# Output goes to editor/dist/ which is embedded via go:embed
```

## Compose Mode

The chat interface includes a `/compose` command that lets you design pipelines conversationally. Instead of manually wiring nodes in the visual editor, describe your use case and the LLM builds the pipeline config for you.

### Usage

1. Open the chat interface and type `/compose`
2. Describe your workflow (e.g. "I need to route customer questions to different specialists")
3. Iterate on the design through conversation — the assistant suggests node types, edges, and tools
4. Type `/done` when satisfied — the assistant outputs the final pipeline as JSON
5. Click **Save & Use** to save the pipeline and switch to it immediately

### Available Node Types in Compose

| Type           | Use when you need to...        |
| -------------- | ------------------------------ |
| `llm`          | Generate text                  |
| `worker`       | Execute tasks with tool access |
| `router`       | Classify and route to branches |
| `gate`         | Validate before proceeding     |
| `orchestrator` | Dynamically decompose tasks    |
| `coordinator`  | Distribute work to nodes       |
| `aggregator`   | Combine outputs from nodes     |
| `synthesizer`  | Synthesize inputs into one     |
| `evaluator`    | Score quality / trigger loops  |

### Edge Types

| Type          | Description                            |
| ------------- | -------------------------------------- |
| `direct`      | Standard sequential flow               |
| `conditional` | Router picks one path                  |
| `dynamic`     | Orchestrator picks a subset of workers |
| `feedback`    | Loop back for iterative refinement     |

## Environment Variables

| Variable            | Description                                         |
| ------------------- | --------------------------------------------------- |
| `OPENAI_API_KEY`    | OpenAI API key                                      |
| `ANTHROPIC_API_KEY` | Anthropic API key                                   |
| `OLLAMA_URL`        | Ollama server URL (default: http://localhost:11434) |
| `DATABASE_URL`      | PostgreSQL DSN for pgvector (optional)              |
| `FISSIO_DATA_DIR`   | Data directory for SQLite (default: ./data)         |

## Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                         GO-FISSIO                              │
│                                                                │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Pipeline  │    │   Engine    │    │   Server    │         │
│  │   Config    │───▶│  Executor   │◀──▶│   API       │         │
│  └─────────────┘    └──────┬──────┘    └─────────────┘         │
│                            │                                   │
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
└────────────────────────────────────────────────────────────────┘
```

## Package Structure

| Package     | Description                              |
| ----------- | ---------------------------------------- |
| `config`    | Pipeline schema, node/edge types         |
| `core`      | Error types, messages, model config      |
| `engine`    | DAG execution engine                     |
| `llm`       | LLM provider clients + embeddings        |
| `tools`     | Tool registry and built-in tools         |
| `vector`    | Vector store interface + implementations |
| `pipelines` | Pre-built pipeline templates             |
| `server`    | HTTP server with SSE                     |

## LLM Providers

| Provider  | Chat | Embeddings | Default Model            |
| --------- | ---- | ---------- | ------------------------ |
| OpenAI    | Yes  | Yes        | `text-embedding-3-small` |
| Anthropic | Yes  | No         | —                        |
| Ollama    | Yes  | Yes        | `nomic-embed-text`       |

## Node Types

| Type           | Description           | Tools |
| -------------- | --------------------- | ----- |
| `llm`          | Simple LLM call       | No    |
| `worker`       | Agentic tool loop     | Yes   |
| `router`       | Classifies and routes | No    |
| `gate`         | Validates input       | No    |
| `aggregator`   | Combines outputs      | No    |
| `orchestrator` | Dynamic decomposition | No    |
| `evaluator`    | Quality scoring       | No    |
| `synthesizer`  | Synthesizes inputs    | No    |
| `coordinator`  | Distributes work      | No    |

## Built-in Tools

| Tool                | Description                       |
| ------------------- | --------------------------------- |
| `fetch_url`         | Fetches content from a URL        |
| `web_search`        | Web search via Tavily API         |
| `similarity_search` | Semantic search over vector store |
| `index_document`    | Index documents into vector store |

## RAG (Retrieval-Augmented Generation)

RAG augments LLM responses with context retrieved from your own documents.

![RAG Example](examples/rag/rag-example.png)

## Vector Store Options

**In-memory** (default, no dependencies):

```go
store := vector.NewMemoryStore()
```

**PostgreSQL with pgvector** (production):

```go
store, err := vector.NewPgVector(ctx, "postgres://user:pass@localhost/db")
```

## Embedding Models

| Provider | Model                    | Dimensions |
| -------- | ------------------------ | ---------- |
| OpenAI   | `text-embedding-3-small` | 1536       |
| OpenAI   | `text-embedding-3-large` | 3072       |
| Ollama   | `nomic-embed-text`       | 768        |

## License

MIT
