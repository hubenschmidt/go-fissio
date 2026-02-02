package vector

import (
	"context"
	"sort"
	"sync"
)

// MemoryStore is an in-memory vector store for development and testing.
type MemoryStore struct {
	mu   sync.RWMutex
	docs map[string]Document
}

// NewMemoryStore creates a new in-memory vector store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		docs: make(map[string]Document),
	}
}

// Upsert stores documents, updating existing ones by ID.
func (s *MemoryStore) Upsert(ctx context.Context, docs []Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, doc := range docs {
		s.docs[doc.ID] = doc
	}
	return nil
}

// Search finds documents similar to the given embedding using brute-force cosine similarity.
func (s *MemoryStore) Search(ctx context.Context, embedding []float64, topK int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := s.computeSimilarities(embedding)

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > 0 && len(results) > topK {
		results = results[:topK]
	}

	return results, nil
}

func (s *MemoryStore) computeSimilarities(embedding []float64) []SearchResult {
	results := make([]SearchResult, 0, len(s.docs))
	for _, doc := range s.docs {
		if len(doc.Embedding) > 0 {
			results = append(results, SearchResult{Document: doc, Score: CosineSimilarity(embedding, doc.Embedding)})
		}
	}
	return results
}

// Delete removes documents by ID.
func (s *MemoryStore) Delete(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.docs, id)
	}
	return nil
}

// Close is a no-op for in-memory store.
func (s *MemoryStore) Close() error {
	return nil
}

// Count returns the number of documents in the store.
func (s *MemoryStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs)
}
