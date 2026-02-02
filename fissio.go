// Package fissio provides a declarative agent framework for building LLM pipelines.
//
// Example usage:
//
//	cfg := config.NewPipeline("research", "Research Pipeline").
//	    Node("researcher", config.NodeWorker).
//	        Prompt("You are a research assistant.").
//	        Tools("web_search", "fetch_url").
//	        Done().
//	    Node("summarizer", config.NodeLLM).
//	        Prompt("Summarize findings.").
//	        Model("gpt-4").
//	        Done().
//	    Edge("researcher", "summarizer").
//	    Build()
//
//	client := llm.NewUnifiedClient(llm.UnifiedConfig{OpenAIKey: os.Getenv("OPENAI_API_KEY")})
//	eng := engine.NewEngine(cfg, engine.EngineConfig{Client: client})
//	result, err := eng.Run(ctx, "Research quantum computing")
package fissio

import (
	"net/http"

	"github.com/hubenschmidt/go-fissio/config"
	"github.com/hubenschmidt/go-fissio/core"
	"github.com/hubenschmidt/go-fissio/editor"
	"github.com/hubenschmidt/go-fissio/engine"
	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/monitor"
	"github.com/hubenschmidt/go-fissio/server"
	"github.com/hubenschmidt/go-fissio/tools"
	"github.com/hubenschmidt/go-fissio/vector"
)

// Re-export node types for convenience
const (
	NodeLLM          = config.NodeLLM
	NodeWorker       = config.NodeWorker
	NodeRouter       = config.NodeRouter
	NodeGate         = config.NodeGate
	NodeAggregator   = config.NodeAggregator
	NodeOrchestrator = config.NodeOrchestrator
	NodeEvaluator    = config.NodeEvaluator
	NodeSynthesizer  = config.NodeSynthesizer
	NodeCoordinator  = config.NodeCoordinator
)

// Builder aliases
type (
	PipelineBuilder = config.PipelineBuilder
	NodeBuilder     = config.NodeBuilder
	PipelineConfig  = config.PipelineConfig
)

// NewPipeline creates a new pipeline builder.
func NewPipeline(id, name string) *PipelineBuilder {
	return config.NewPipeline(id, name)
}

// Engine aliases
type (
	Engine       = engine.Engine
	EngineConfig = engine.EngineConfig
	EngineOutput = engine.EngineOutput
)

// NewEngine creates a new pipeline execution engine.
func NewEngine(pipeline *PipelineConfig, cfg EngineConfig) *Engine {
	return engine.NewEngine(pipeline, cfg)
}

// LLM client aliases
type (
	LLMClient     = llm.Client
	UnifiedClient = llm.UnifiedClient
	UnifiedConfig = llm.UnifiedConfig
)

// NewUnifiedClient creates a new unified LLM client that auto-routes to the appropriate provider.
func NewUnifiedClient(cfg UnifiedConfig) *UnifiedClient {
	return llm.NewUnifiedClient(cfg)
}

// Tool aliases
type (
	Tool         = tools.Tool
	ToolRegistry = tools.Registry
)

// RegisterTool registers a tool with the default registry.
func RegisterTool(t Tool) {
	tools.Register(t)
}

// GetTool retrieves a tool from the default registry.
func GetTool(name string) (Tool, bool) {
	return tools.Get(name)
}

// Core type aliases
type (
	Message     = core.Message
	MessageRole = core.MessageRole
	ToolCall    = core.ToolCall
	ToolResult  = core.ToolResult
	ModelConfig = core.ModelConfig
	AgentError  = core.AgentError
)

// Monitor aliases
type (
	MetricsCollector  = monitor.MetricsCollector
	InMemoryCollector = monitor.InMemoryCollector
	PipelineMetrics   = monitor.PipelineMetrics
)

// NewInMemoryCollector creates a new in-memory metrics collector.
func NewInMemoryCollector(pipelineID string) *InMemoryCollector {
	return monitor.NewInMemoryCollector(pipelineID)
}

// Server aliases
type (
	Server       = server.Server
	ServerConfig = server.Config
)

// NewServer creates a new API server.
func NewServer(cfg ServerConfig) (*Server, error) {
	return server.New(cfg)
}

// EditorHandler returns an http.Handler that serves the embedded editor UI.
func EditorHandler() http.Handler {
	return editor.Handler()
}

// Vector store aliases
type (
	VectorStore        = vector.Store
	VectorDocument     = vector.Document
	VectorSearchResult = vector.SearchResult
)

// NewMemoryVectorStore creates a new in-memory vector store.
func NewMemoryVectorStore() *vector.MemoryStore {
	return vector.NewMemoryStore()
}

// NewPgVectorStore creates a new pgvector-based vector store.
func NewPgVectorStore(dsn string, dimension int) (*vector.PgVectorStore, error) {
	return vector.NewPgVectorStore(dsn, dimension)
}
