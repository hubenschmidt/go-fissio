package server

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/server/store"
	"github.com/hubenschmidt/go-fissio/tools"
	"github.com/hubenschmidt/go-fissio/vector"
)

// Config configures a new Server instance.
type Config struct {
	Client      llm.Client
	Registry    *tools.Registry
	Models      []ModelInfo
	Templates   []PipelineInfo
	OllamaURL   string // Optional: URL for Ollama model discovery
	DatabaseDSN string // Optional: database connection string (postgres:// or sqlite path)

	// Vector store configuration
	VectorStore vector.Store // Optional: inject custom vector store
	EmbedModel  string       // Embedding model (default: text-embedding-3-small)
}

// Server is an HTTP server for the fissio agent framework.
type Server struct {
	client      llm.Client
	registry    *tools.Registry
	models      []ModelInfo
	templates   []PipelineInfo
	pipelines   store.PipelineStore
	traces      store.TraceStore
	vectorStore vector.Store
}

// New creates a new Server with the given configuration.
func New(cfg Config) (*Server, error) {
	registry := cfg.Registry
	if registry == nil {
		registry = tools.DefaultRegistry
	}

	models := cfg.Models
	if len(models) == 0 {
		models = defaultModels()
	}

	// Discover Ollama models if URL provided
	if cfg.OllamaURL != "" {
		ollamaModels, err := llm.DiscoverOllamaModels(cfg.OllamaURL)
		if err != nil {
			log.Printf("[ollama] Discovery failed (is Ollama running?): %v", err)
		} else {
			log.Printf("[ollama] Found %d local models", len(ollamaModels))
			for _, m := range ollamaModels {
				log.Printf("[ollama]   - %s (%s)", m.Name, m.ID)
				models = append(models, ModelInfo{
					ID:      m.ID,
					Name:    m.Name,
					Model:   m.Model,
					APIBase: m.APIBase,
				})
			}
		}
	}

	templates := cfg.Templates
	if len(templates) == 0 {
		templates = defaultTemplates()
	}

	// Initialize database stores
	traceStore, pipelineStore, err := store.NewStores(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("initialize stores: %w", err)
	}

	log.Printf("[store] Initialized database storage")

	// Initialize vector store
	var vectorStore vector.Store
	if cfg.VectorStore != nil {
		vectorStore = cfg.VectorStore
	} else if strings.HasPrefix(cfg.DatabaseDSN, "postgres://") || strings.HasPrefix(cfg.DatabaseDSN, "postgresql://") {
		vs, err := vector.NewPgVectorStore(cfg.DatabaseDSN, 1536)
		if err != nil {
			log.Printf("[vector] Failed to initialize pgvector: %v", err)
		} else {
			vectorStore = vs
			log.Printf("[vector] Initialized pgvector store")
		}
	}
	if vectorStore == nil {
		vectorStore = vector.NewMemoryStore()
		log.Printf("[vector] Using in-memory vector store")
	}

	// Register semantic search tools if we have an embedding client
	if embedder, ok := cfg.Client.(llm.EmbeddingClient); ok {
		embedModel := cfg.EmbedModel
		if embedModel == "" {
			embedModel = "text-embedding-3-small"
		}
		registry.Register(tools.NewSimilaritySearchTool(vectorStore, embedder, embedModel))
		registry.Register(tools.NewIndexDocumentTool(vectorStore, embedder, embedModel))
		log.Printf("[vector] Registered similarity_search and index_document tools (model: %s)", embedModel)
	}

	return &Server{
		client:      cfg.Client,
		registry:    registry,
		models:      models,
		templates:   templates,
		pipelines:   pipelineStore,
		traces:      traceStore,
		vectorStore: vectorStore,
	}, nil
}

// Close closes the server and releases resources.
func (s *Server) Close() error {
	var errs []error
	if err := s.traces.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.pipelines.Close(); err != nil {
		errs = append(errs, err)
	}
	if s.vectorStore != nil {
		if err := s.vectorStore.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("close stores: %v", errs)
	}
	return nil
}

// Handler returns an http.Handler for the API routes.
// All routes are prefixed with /api/.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /init", s.handleInit)
	mux.HandleFunc("GET /tools", s.handleTools)
	mux.HandleFunc("POST /chat", s.handleChat)

	mux.HandleFunc("GET /pipelines", s.handlePipelineList)
	mux.HandleFunc("POST /pipelines/save", s.handlePipelineSave)
	mux.HandleFunc("POST /pipelines/delete", s.handlePipelineDelete)

	mux.HandleFunc("GET /api/traces", s.handleTraceList)
	mux.HandleFunc("GET /api/traces/{id}", s.handleTraceGet)
	mux.HandleFunc("DELETE /api/traces/{id}", s.handleTraceDelete)
	mux.HandleFunc("GET /api/metrics/summary", s.handleMetricsSummary)

	return corsMiddleware(mux)
}

func defaultModels() []ModelInfo {
	return []ModelInfo{
		{ID: "openai-gpt5", Name: "GPT-5.2 (OpenAI)", Model: "gpt-5.2-2025-12-11"},
		{ID: "openai-codex", Name: "GPT-5.2 Codex (OpenAI)", Model: "gpt-5.2-codex"},
		{ID: "anthropic-opus", Name: "Claude Opus 4.5 (Anthropic)", Model: "claude-opus-4-5-20251101"},
		{ID: "anthropic-sonnet", Name: "Claude Sonnet 4.5 (Anthropic)", Model: "claude-sonnet-4-5-20250929"},
		{ID: "anthropic-haiku", Name: "Claude Haiku 4.5 (Anthropic)", Model: "claude-haiku-4-5-20251001"},
	}
}

func defaultTemplates() []PipelineInfo {
	llmType := "llm"
	workerType := "worker"
	routerType := "router"
	defaultPrompt := "You are a helpful assistant."
	researchPrompt := "You are a research assistant. Search the web for information."
	routerPrompt := "Classify the user's request and route to the appropriate specialist."

	return []PipelineInfo{
		{
			ID:          "simple-chat",
			Name:        "Simple Chat",
			Description: "Single LLM node for basic chat",
			Nodes: []NodeInfo{
				{ID: "assistant", NodeType: llmType, Prompt: &defaultPrompt},
			},
			Edges: []EdgeInfo{},
		},
		{
			ID:          "research-agent",
			Name:        "Research Agent",
			Description: "Worker node with web search tools",
			Nodes: []NodeInfo{
				{ID: "researcher", NodeType: workerType, Prompt: &researchPrompt, Tools: []string{"web_search", "fetch_url"}},
			},
			Edges: []EdgeInfo{},
		},
		{
			ID:          "router-pipeline",
			Name:        "Router Pipeline",
			Description: "Route requests to specialized agents",
			Nodes: []NodeInfo{
				{ID: "router", NodeType: routerType, Prompt: &routerPrompt},
				{ID: "coder", NodeType: llmType, Prompt: strPtr("You are a coding expert.")},
				{ID: "writer", NodeType: llmType, Prompt: strPtr("You are a writing expert.")},
			},
			Edges: []EdgeInfo{},
		},
	}
}

func strPtr(s string) *string {
	return &s
}
