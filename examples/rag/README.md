# Interactive RAG Demo

An interactive CLI demonstrating go-fissio's semantic search and RAG capabilities.

## Features

- **Semantic search** over indexed documents using pgvector
- **RAG pipeline** that retrieves context before generating answers
- **Interactive REPL** with commands to search, index, and query

## Quick Start

```bash
cd examples/rag

# Configure environment
cp .example.env .env
# Edit .env and add your OpenAI API key

# Build and start containers
docker compose up -d --build

# Attach to the CLI (prompt already printed, just type)
docker attach rag-example
```

**Tips:**
- Detach without stopping: `Ctrl+P`, `Ctrl+Q`
- Rebuild after code changes: `docker compose up -d --build`
- View logs: `docker compose logs -f rag-example`

## Running Locally (Alternative)

```bash
cd examples/rag
docker compose up -d postgres  # Just postgres

cd ../..
source examples/rag/.env
go run ./examples/rag
```

## Commands

| Command | Description |
|---------|-------------|
| `<question>` | Ask a question (runs RAG pipeline) |
| `/search <query>` | Show similarity search results |
| `/index <content>` | Add a document to the knowledge base |
| `/list` | Show indexed documents |
| `/verbose` | Toggle verbose mode (show retrieved context) |
| `/help` | Show available commands |
| `/quit` | Exit |

## Example Session

```
=== RAG Demo ===

Connecting to PostgreSQL...
Connected.

Indexing sample documents...
Indexed 7 documents.

Commands:
  <question>        Ask a question (runs RAG pipeline)
  /search <query>   Show similarity search results
  /index <content>  Add a document to the knowledge base
  /list             Show indexed documents
  /verbose          Toggle verbose mode
  /help             Show this help
  /quit             Exit

> What tables store customer information?

Retrieving context...
Found 3 relevant documents:
  - table-users (score: 0.847)
  - doc-customer-lifetime (score: 0.723)
  - table-orders (score: 0.698)

Generating response...

The `users` table stores customer account information, including:
- id (UUID, primary key)
- email (VARCHAR, unique)
- name (VARCHAR)
- created_at (TIMESTAMP)
- last_login (TIMESTAMP)

[Tokens: 892 in, 156 out]

> /search revenue

Searching for: revenue
1. [0.892] doc-revenue-query
   To calculate total revenue, join orders with order_items: SELECT SUM(oi.quantity * oi.unit_price)...

2. [0.734] table-orders
   The `orders` table tracks customer purchases. Columns: id (UUID, primary key), user_id (UUID, fo...

> /index The payments table stores payment transactions with columns: id, order_id, amount, method, status
Indexed document: user-doc-101

> How do I process payments?

Retrieving context...
Found 3 relevant documents:
  - user-doc-101 (score: 0.856)
  - table-orders (score: 0.712)
  - doc-revenue-query (score: 0.654)

Generating response...

Based on the payments table, you can process payments by...

> /quit
Goodbye!
```

## How It Works

1. **Startup**: Connects to PostgreSQL and indexes sample documents (e-commerce schema)
2. **Indexing**: Each document is embedded using OpenAI's `text-embedding-3-small` model
3. **Search**: Queries are embedded and compared using cosine similarity (pgvector)
4. **RAG**: Retrieved documents are passed as context to the LLM for generation

## Customization

- **Add your own documents**: Use `/index <content>` or modify `sampleDocs` in main.go
- **Change embedding model**: Modify `embedModel` variable
- **Adjust retrieval count**: Modify the `topK` parameter in search calls
