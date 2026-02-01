package store

import (
	"fmt"
	"strings"
)

// NewStores creates trace and pipeline stores based on the DSN.
// - Empty DSN: SQLite at data/fissio.db
// - postgres:// or postgresql://: PostgreSQL
// - Anything else: SQLite at the specified path
func NewStores(dsn string) (TraceStore, PipelineStore, error) {
	if dsn == "" {
		return NewSQLiteStores("data/fissio.db")
	}

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		ts, ps, err := NewPostgresStores(dsn)
		if err != nil {
			return nil, nil, fmt.Errorf("postgres: %w", err)
		}
		return ts, ps, nil
	}

	return NewSQLiteStores(dsn)
}
