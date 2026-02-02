package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/vector"
)

// SimilaritySearchTool searches a vector store for similar documents.
type SimilaritySearchTool struct {
	store    vector.Store
	embedder llm.EmbeddingClient
	model    string
}

// NewSimilaritySearchTool creates a new similarity search tool.
func NewSimilaritySearchTool(store vector.Store, embedder llm.EmbeddingClient, model string) *SimilaritySearchTool {
	return &SimilaritySearchTool{
		store:    store,
		embedder: embedder,
		model:    model,
	}
}

func (t *SimilaritySearchTool) Name() string {
	return "similarity_search"
}

func (t *SimilaritySearchTool) Description() string {
	return "Search for documents similar to a query using semantic similarity. Returns the most relevant documents."
}

func (t *SimilaritySearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The search query to find similar documents"
			},
			"top_k": {
				"type": "integer",
				"description": "Maximum number of results to return (default: 5)"
			}
		},
		"required": ["query"]
	}`)
}

func (t *SimilaritySearchTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var req struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	// Generate embedding for query
	resp, err := t.embedder.Embed(ctx, t.model, req.Query)
	if err != nil {
		return "", fmt.Errorf("embed query: %w", err)
	}

	// Search vector store
	results, err := t.store.Search(ctx, resp.Embedding, req.TopK)
	if err != nil {
		return "", fmt.Errorf("search: %w", err)
	}

	if len(results) == 0 {
		return "No similar documents found.", nil
	}

	return formatSearchResults(results), nil
}

func formatSearchResults(results []vector.SearchResult) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d relevant documents:\n\n", len(results)))

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("--- Document %d (score: %.3f) ---\n", i+1, r.Score))
		sb.WriteString(r.Document.Content)
		sb.WriteString("\n\n")
	}

	return sb.String()
}
