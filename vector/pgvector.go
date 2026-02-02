package vector

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PgVectorStore is a PostgreSQL-based vector store using pgvector.
type PgVectorStore struct {
	db        *sql.DB
	dimension int
}

// NewPgVectorStore creates a new pgvector-based store.
// The dimension parameter specifies the embedding dimension (e.g., 1536 for OpenAI).
func NewPgVectorStore(dsn string, dimension int) (*PgVectorStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	store := &PgVectorStore{db: db, dimension: dimension}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return store, nil
}

func (s *PgVectorStore) migrate() error {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			content TEXT NOT NULL,
			embedding vector(%d),
			metadata JSONB DEFAULT '{}',
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`, s.dimension),
		`CREATE INDEX IF NOT EXISTS idx_documents_embedding ON documents USING hnsw (embedding vector_cosine_ops)`,
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("execute migration: %w", err)
		}
	}

	return nil
}

// Upsert stores documents, updating existing ones by ID.
func (s *PgVectorStore) Upsert(ctx context.Context, docs []Document) error {
	for _, doc := range docs {
		metadata, err := json.Marshal(doc.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}

		embeddingStr := formatEmbedding(doc.Embedding)

		_, err = s.db.ExecContext(ctx, `
			INSERT INTO documents (id, content, embedding, metadata)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE SET
				content = EXCLUDED.content,
				embedding = EXCLUDED.embedding,
				metadata = EXCLUDED.metadata
		`, doc.ID, doc.Content, embeddingStr, metadata)
		if err != nil {
			return fmt.Errorf("upsert document: %w", err)
		}
	}
	return nil
}

// Search finds documents similar to the given embedding.
func (s *PgVectorStore) Search(ctx context.Context, embedding []float64, topK int) ([]SearchResult, error) {
	embeddingStr := formatEmbedding(embedding)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, content, embedding, metadata, 1 - (embedding <=> $1) AS score
		FROM documents
		ORDER BY embedding <=> $1
		LIMIT $2
	`, embeddingStr, topK)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var doc Document
		var embeddingStr string
		var metadataBytes []byte
		var score float64

		if err := rows.Scan(&doc.ID, &doc.Content, &embeddingStr, &metadataBytes, &score); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		doc.Embedding = parseEmbedding(embeddingStr)
		if len(metadataBytes) > 0 {
			json.Unmarshal(metadataBytes, &doc.Metadata)
		}

		results = append(results, SearchResult{
			Document: doc,
			Score:    score,
		})
	}

	return results, rows.Err()
}

// Delete removes documents by ID.
func (s *PgVectorStore) Delete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM documents WHERE id IN (%s)", strings.Join(placeholders, ","))
	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

// Close closes the database connection.
func (s *PgVectorStore) Close() error {
	return s.db.Close()
}

// formatEmbedding converts a float64 slice to pgvector format: "[0.1,0.2,0.3]"
func formatEmbedding(embedding []float64) string {
	if len(embedding) == 0 {
		return ""
	}

	parts := make([]string, len(embedding))
	for i, v := range embedding {
		parts[i] = fmt.Sprintf("%g", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// parseEmbedding converts pgvector format back to float64 slice.
func parseEmbedding(s string) []float64 {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	result := make([]float64, len(parts))
	for i, p := range parts {
		fmt.Sscanf(p, "%f", &result[i])
	}
	return result
}
