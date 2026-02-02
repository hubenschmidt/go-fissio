package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/vector"
)

// IndexDocumentTool adds documents to the vector store.
type IndexDocumentTool struct {
	store    vector.Store
	embedder llm.EmbeddingClient
	model    string
}

// NewIndexDocumentTool creates a new index document tool.
func NewIndexDocumentTool(store vector.Store, embedder llm.EmbeddingClient, model string) *IndexDocumentTool {
	return &IndexDocumentTool{
		store:    store,
		embedder: embedder,
		model:    model,
	}
}

func (t *IndexDocumentTool) Name() string {
	return "index_document"
}

func (t *IndexDocumentTool) Description() string {
	return "Add a document to the knowledge base for future similarity searches."
}

func (t *IndexDocumentTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"id": {
				"type": "string",
				"description": "Unique identifier for the document"
			},
			"content": {
				"type": "string",
				"description": "The document content to index"
			},
			"metadata": {
				"type": "object",
				"description": "Optional metadata to store with the document"
			}
		},
		"required": ["id", "content"]
	}`)
}

func (t *IndexDocumentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var req struct {
		ID       string         `json:"id"`
		Content  string         `json:"content"`
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal(args, &req); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}

	// Generate embedding
	resp, err := t.embedder.Embed(ctx, t.model, req.Content)
	if err != nil {
		return "", fmt.Errorf("embed content: %w", err)
	}

	// Store document
	doc := vector.Document{
		ID:        req.ID,
		Content:   req.Content,
		Embedding: resp.Embedding,
		Metadata:  req.Metadata,
	}

	if err := t.store.Upsert(ctx, []vector.Document{doc}); err != nil {
		return "", fmt.Errorf("upsert: %w", err)
	}

	return fmt.Sprintf("Document '%s' indexed successfully.", req.ID), nil
}
