// Package vector provides vector storage and similarity search.
package vector

import "context"

// Document represents a document with optional embedding.
type Document struct {
	ID        string         `json:"id"`
	Content   string         `json:"content"`
	Embedding []float64      `json:"embedding,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// SearchResult represents a search result with similarity score.
type SearchResult struct {
	Document Document `json:"document"`
	Score    float64  `json:"score"` // cosine similarity (0-1)
}

// Store provides vector storage and similarity search operations.
type Store interface {
	// Upsert stores documents, updating existing ones by ID.
	Upsert(ctx context.Context, docs []Document) error

	// Search finds documents similar to the given embedding.
	Search(ctx context.Context, embedding []float64, topK int) ([]SearchResult, error)

	// Delete removes documents by ID.
	Delete(ctx context.Context, ids []string) error

	// Close releases resources.
	Close() error
}
