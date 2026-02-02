// Interactive RAG demo using go-fissio's semantic search capabilities.
//
// This example:
// 1. Connects to a PostgreSQL database with pgvector
// 2. Indexes sample documents about a fictional e-commerce database
// 3. Provides an interactive CLI to query using RAG
//
// Commands:
//   <question>        - Ask a question (runs RAG pipeline)
//   /search <query>   - Show raw similarity search results
//   /index <content>  - Add a document to the vector store
//   /list             - Show indexed document IDs
//   /verbose          - Toggle verbose mode (show retrieved context)
//   /help             - Show available commands
//   /quit or /q       - Exit
//
// Environment variables:
//   - OPENAI_API_KEY: Required for embeddings and chat
//   - DATABASE_URL: PostgreSQL connection string
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hubenschmidt/go-fissio"
	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/pipelines"
	"github.com/hubenschmidt/go-fissio/tools"
	"github.com/hubenschmidt/go-fissio/vector"
)

var (
	verbose    = true
	embedModel = "text-embedding-3-small"
	docCounter = 100
)

// Sample documents describing a fictional e-commerce database schema
var sampleDocs = []vector.Document{
	{
		ID:      "table-users",
		Content: "The `users` table stores customer account information. Columns: id (UUID, primary key), email (VARCHAR, unique), name (VARCHAR), created_at (TIMESTAMP), last_login (TIMESTAMP). Indexes on email for fast lookups.",
	},
	{
		ID:      "table-orders",
		Content: "The `orders` table tracks customer purchases. Columns: id (UUID, primary key), user_id (UUID, foreign key to users), total_amount (DECIMAL), status (ENUM: pending, paid, shipped, delivered, cancelled), created_at (TIMESTAMP). Foreign key relationship: orders.user_id -> users.id",
	},
	{
		ID:      "table-products",
		Content: "The `products` table contains the product catalog. Columns: id (UUID, primary key), name (VARCHAR), description (TEXT), price (DECIMAL), category_id (UUID, foreign key), stock_quantity (INTEGER), created_at (TIMESTAMP).",
	},
	{
		ID:      "table-order_items",
		Content: "The `order_items` table is a junction table linking orders to products. Columns: id (UUID), order_id (UUID, foreign key to orders), product_id (UUID, foreign key to products), quantity (INTEGER), unit_price (DECIMAL). This table enables many-to-many relationship between orders and products.",
	},
	{
		ID:      "table-categories",
		Content: "The `categories` table organizes products into hierarchical categories. Columns: id (UUID, primary key), name (VARCHAR), parent_id (UUID, self-referential foreign key for nested categories), description (TEXT).",
	},
	{
		ID:      "doc-revenue-query",
		Content: "To calculate total revenue, join orders with order_items: SELECT SUM(oi.quantity * oi.unit_price) as revenue FROM orders o JOIN order_items oi ON o.id = oi.order_id WHERE o.status = 'delivered'. Filter by date range using o.created_at.",
	},
	{
		ID:      "doc-customer-lifetime",
		Content: "Customer lifetime value (CLV) query: SELECT u.id, u.email, SUM(o.total_amount) as lifetime_value, COUNT(o.id) as order_count FROM users u LEFT JOIN orders o ON u.id = o.user_id WHERE o.status = 'delivered' GROUP BY u.id, u.email ORDER BY lifetime_value DESC.",
	},
}

func main() {
	ctx := context.Background()

	// Get configuration
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/fissio?sslmode=disable"
	}

	fmt.Println("=== RAG Demo ===")
	fmt.Println()

	log.Printf("[init] Embedding model: %s (1536 dimensions)", embedModel)

	// Create unified client
	log.Printf("[init] Creating OpenAI client...")
	client := fissio.NewUnifiedClient(fissio.UnifiedConfig{
		OpenAIKey: apiKey,
	})
	log.Printf("[init] OpenAI client ready")

	// Create vector store
	log.Printf("[vector] Connecting to PostgreSQL with pgvector...")
	store, err := vector.NewPgVectorStore(databaseURL, 1536)
	if err != nil {
		log.Fatalf("Failed to create vector store: %v", err)
	}
	defer store.Close()
	log.Printf("[vector] Connected to pgvector (cosine similarity enabled)")

	// Register similarity_search tool
	searchTool := tools.NewSimilaritySearchTool(store, client, embedModel)
	fissio.RegisterTool(searchTool)
	log.Printf("[tools] Registered similarity_search tool")

	// Index sample documents
	fmt.Println("Indexing sample documents...")
	if err := indexDocuments(ctx, client, store, sampleDocs); err != nil {
		log.Fatalf("Failed to index documents: %v", err)
	}
	fmt.Printf("Indexed %d documents.\n", len(sampleDocs))
	fmt.Println()

	// Create RAG pipeline
	pipeline := pipelines.NewSimpleRAGPipeline(
		"You are a helpful database assistant. Answer questions about the e-commerce database schema. " +
			"Use the provided context to give accurate answers. If the context doesn't contain enough information, say so.",
	)

	engine := fissio.NewEngine(pipeline, fissio.EngineConfig{
		Client: client,
	})

	// Print help
	printHelp()
	fmt.Println()

	// REPL
	scanner := bufio.NewScanner(os.Stdin)
	running := true
	for running {
		fmt.Print("> ")
		if !scanner.Scan() {
			running = false
		} else {
			running = processInput(ctx, strings.TrimSpace(scanner.Text()), client, store, engine)
		}
	}

	fmt.Println("Goodbye!")
}

func processInput(ctx context.Context, input string, client *llm.UnifiedClient, store *vector.PgVectorStore, engine *fissio.Engine) bool {
	if input == "" {
		return true
	}

	if strings.HasPrefix(input, "/") {
		return !handleCommand(ctx, input, client, store)
	}

	runRAGQuery(ctx, input, client, store, engine)
	return true
}

type commandHandler func(ctx context.Context, arg string, client *llm.UnifiedClient, store *vector.PgVectorStore) bool

func handleCommand(ctx context.Context, input string, client *llm.UnifiedClient, store *vector.PgVectorStore) bool {
	parts := strings.SplitN(input, " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	handlers := map[string]commandHandler{
		"/quit":    cmdQuit,
		"/q":       cmdQuit,
		"/help":    cmdHelp,
		"/h":       cmdHelp,
		"/verbose": cmdVerbose,
		"/v":       cmdVerbose,
		"/search":  cmdSearch,
		"/s":       cmdSearch,
		"/index":   cmdIndex,
		"/i":       cmdIndex,
		"/list":    cmdList,
		"/l":       cmdList,
	}

	handler, ok := handlers[cmd]
	if !ok {
		fmt.Printf("Unknown command: %s (type /help for commands)\n", cmd)
		return false
	}
	return handler(ctx, arg, client, store)
}

func cmdQuit(_ context.Context, _ string, _ *llm.UnifiedClient, _ *vector.PgVectorStore) bool {
	return true
}

func cmdHelp(_ context.Context, _ string, _ *llm.UnifiedClient, _ *vector.PgVectorStore) bool {
	printHelp()
	return false
}

func cmdVerbose(_ context.Context, _ string, _ *llm.UnifiedClient, _ *vector.PgVectorStore) bool {
	verbose = !verbose
	fmt.Printf("Verbose mode: %v\n", verbose)
	return false
}

func cmdSearch(ctx context.Context, arg string, client *llm.UnifiedClient, store *vector.PgVectorStore) bool {
	if arg == "" {
		fmt.Println("Usage: /search <query>")
		return false
	}
	searchDocuments(ctx, arg, client, store)
	return false
}

func cmdIndex(ctx context.Context, arg string, client *llm.UnifiedClient, store *vector.PgVectorStore) bool {
	if arg == "" {
		fmt.Println("Usage: /index <content>")
		return false
	}
	indexNewDocument(ctx, arg, client, store)
	return false
}

func cmdList(ctx context.Context, _ string, _ *llm.UnifiedClient, store *vector.PgVectorStore) bool {
	listDocuments(ctx, store)
	return false
}

func printHelp() {
	fmt.Println("Commands:")
	fmt.Println("  <question>        Ask a question (runs RAG pipeline)")
	fmt.Println("  /search <query>   Show similarity search results")
	fmt.Println("  /index <content>  Add a document to the knowledge base")
	fmt.Println("  /list             Show indexed documents")
	fmt.Println("  /verbose          Toggle verbose mode")
	fmt.Println("  /help             Show this help")
	fmt.Println("  /quit             Exit")
}

func indexDocuments(ctx context.Context, client *llm.UnifiedClient, store *vector.PgVectorStore, docs []vector.Document) error {
	for i := range docs {
		log.Printf("[embed] Generating embedding for %s (%d chars)...", docs[i].ID, len(docs[i].Content))
		resp, err := client.Embed(ctx, embedModel, docs[i].Content)
		if err != nil {
			return fmt.Errorf("embed %s: %w", docs[i].ID, err)
		}
		docs[i].Embedding = resp.Embedding
		log.Printf("[embed] Got %d-dimension vector, %d tokens", len(resp.Embedding), resp.TokenCount)

		log.Printf("[vector] Upserting %s to pgvector...", docs[i].ID)
		if err := store.Upsert(ctx, []vector.Document{docs[i]}); err != nil {
			return fmt.Errorf("upsert %s: %w", docs[i].ID, err)
		}
	}
	return nil
}

func searchDocuments(ctx context.Context, query string, client *llm.UnifiedClient, store *vector.PgVectorStore) {
	fmt.Printf("\nSearching for: %s\n", query)

	log.Printf("[search] Embedding query: %q", query)
	resp, err := client.Embed(ctx, embedModel, query)
	if err != nil {
		fmt.Printf("Error embedding query: %v\n", err)
		return
	}
	log.Printf("[search] Query embedded to %d dimensions", len(resp.Embedding))

	log.Printf("[search] Running cosine similarity search (top 5)...")
	results, err := store.Search(ctx, resp.Embedding, 5)
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		return
	}
	log.Printf("[search] Found %d results", len(results))

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	fmt.Println()
	for i, r := range results {
		preview := r.Document.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		fmt.Printf("%d. [%.3f] %s\n", i+1, r.Score, r.Document.ID)
		fmt.Printf("   %s\n", preview)
		fmt.Println()
	}
}

func indexNewDocument(ctx context.Context, content string, client *llm.UnifiedClient, store *vector.PgVectorStore) {
	docCounter++
	id := fmt.Sprintf("user-doc-%d", docCounter)

	log.Printf("[index] Creating new document: %s", id)
	log.Printf("[index] Content: %q", truncate(content, 80))
	log.Printf("[index] Generating embedding...")

	resp, err := client.Embed(ctx, embedModel, content)
	if err != nil {
		fmt.Printf("Error embedding document: %v\n", err)
		return
	}
	log.Printf("[index] Embedded to %d dimensions (%d tokens)", len(resp.Embedding), resp.TokenCount)

	doc := vector.Document{
		ID:        id,
		Content:   content,
		Embedding: resp.Embedding,
	}

	log.Printf("[index] Storing in pgvector...")
	if err := store.Upsert(ctx, []vector.Document{doc}); err != nil {
		fmt.Printf("Error indexing document: %v\n", err)
		return
	}

	fmt.Printf("Indexed document: %s\n", id)
}

func listDocuments(ctx context.Context, store *vector.PgVectorStore) {
	// Search with a generic embedding to list all docs (hacky but works)
	results, err := store.Search(ctx, make([]float64, 1536), 100)
	if err != nil {
		fmt.Printf("Error listing documents: %v\n", err)
		return
	}

	fmt.Printf("Indexed documents (%d):\n", len(results))
	for _, r := range results {
		preview := r.Document.Content
		if len(preview) > 60 {
			preview = preview[:60] + "..."
		}
		fmt.Printf("  - %s: %s\n", r.Document.ID, preview)
	}
}

func runRAGQuery(ctx context.Context, query string, client *llm.UnifiedClient, store *vector.PgVectorStore, engine *fissio.Engine) {
	fmt.Println()
	log.Printf("[rag] Starting RAG pipeline for query: %q", truncate(query, 60))

	// Step 1: Embed query
	log.Printf("[rag] Step 1: Embedding query...")
	resp, err := client.Embed(ctx, embedModel, query)
	if err != nil {
		fmt.Printf("Error embedding query: %v\n", err)
		return
	}
	log.Printf("[rag] Query embedded (%d dimensions)", len(resp.Embedding))

	// Step 2: Semantic search
	log.Printf("[rag] Step 2: Semantic search (top 3)...")
	results, err := store.Search(ctx, resp.Embedding, 3)
	if err != nil {
		fmt.Printf("Error searching: %v\n", err)
		return
	}

	log.Printf("[rag] Found %d relevant documents:", len(results))
	for _, r := range results {
		log.Printf("[rag]   → %s (similarity: %.1f%%)", r.Document.ID, r.Score*100)
	}

	// Step 3: Build augmented prompt with context
	log.Printf("[rag] Step 3: Augmenting prompt with context...")
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Use the following context to answer the question:\n\n")
	for _, r := range results {
		contextBuilder.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", r.Document.ID, r.Document.Content))
	}
	contextBuilder.WriteString("Question: ")
	contextBuilder.WriteString(query)

	augmentedQuery := contextBuilder.String()
	log.Printf("[rag] Context size: %d chars from %d documents", len(augmentedQuery)-len(query), len(results))

	// Step 4: Generate response
	log.Printf("[rag] Step 4: Generating response with LLM...")
	result, err := engine.Run(ctx, augmentedQuery)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	log.Printf("[rag] Response generated successfully")
	fmt.Println()
	fmt.Println(result.Content)
	fmt.Println()

	// Show token usage
	var tokensIn, tokensOut int
	for _, out := range result.Outputs {
		tokensIn += out.TokensIn
		tokensOut += out.TokensOut
	}
	log.Printf("[rag] Token usage: %d in, %d out", tokensIn, tokensOut)
	fmt.Println()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
