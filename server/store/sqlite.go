package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hubenschmidt/go-fissio/server/store/migrations"
	_ "modernc.org/sqlite"
)

// SQLiteTraceStore implements TraceStore using SQLite
type SQLiteTraceStore struct {
	db *sql.DB
}

// SQLitePipelineStore implements PipelineStore using SQLite
type SQLitePipelineStore struct {
	db *sql.DB
}

// NewSQLiteStores creates SQLite-backed trace and pipeline stores
func NewSQLiteStores(dsn string) (TraceStore, PipelineStore, error) {
	if dsn == "" {
		dsn = "data/fissio.db"
	}

	dir := filepath.Dir(dsn)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, nil, fmt.Errorf("create data directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open sqlite: %w", err)
	}

	if err := runSQLiteMigrations(db); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	return &SQLiteTraceStore{db: db}, &SQLitePipelineStore{db: db}, nil
}

func runSQLiteMigrations(db *sql.DB) error {
	data, err := migrations.SQLite.ReadFile("sqlite/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = db.Exec(string(data))
	if err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}

// TraceStore implementation

func (s *SQLiteTraceStore) Add(ctx context.Context, t TraceInfo) error {
	spans, err := json.Marshal(t.Spans)
	if err != nil {
		return fmt.Errorf("marshal spans: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO traces (
			trace_id, pipeline_id, pipeline_name, timestamp, input, output,
			total_elapsed_ms, total_input_tokens, total_output_tokens,
			total_tool_calls, status, spans
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.TraceID, t.PipelineID, t.PipelineName, t.Timestamp, t.Input, t.Output,
		t.TotalElapsedMs, t.TotalInputTokens, t.TotalOutputTokens,
		t.TotalToolCalls, t.Status, string(spans),
	)
	if err != nil {
		return fmt.Errorf("insert trace: %w", err)
	}
	return nil
}

func (s *SQLiteTraceStore) Get(ctx context.Context, id string) (TraceInfo, error) {
	var t TraceInfo
	var spansJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT trace_id, pipeline_id, pipeline_name, timestamp, input, output,
			   total_elapsed_ms, total_input_tokens, total_output_tokens,
			   total_tool_calls, status, spans
		FROM traces WHERE trace_id = ?`, id).Scan(
		&t.TraceID, &t.PipelineID, &t.PipelineName, &t.Timestamp, &t.Input, &t.Output,
		&t.TotalElapsedMs, &t.TotalInputTokens, &t.TotalOutputTokens,
		&t.TotalToolCalls, &t.Status, &spansJSON,
	)
	if err == sql.ErrNoRows {
		return t, ErrNotFound
	}
	if err != nil {
		return t, fmt.Errorf("query trace: %w", err)
	}

	if err := json.Unmarshal([]byte(spansJSON), &t.Spans); err != nil {
		return t, fmt.Errorf("unmarshal spans: %w", err)
	}
	return t, nil
}

func (s *SQLiteTraceStore) List(ctx context.Context) ([]TraceInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT trace_id, pipeline_id, pipeline_name, timestamp, input, output,
			   total_elapsed_ms, total_input_tokens, total_output_tokens,
			   total_tool_calls, status, spans
		FROM traces ORDER BY timestamp DESC`)
	if err != nil {
		return nil, fmt.Errorf("query traces: %w", err)
	}
	defer rows.Close()

	var traces []TraceInfo
	for rows.Next() {
		var t TraceInfo
		var spansJSON string
		if err := rows.Scan(
			&t.TraceID, &t.PipelineID, &t.PipelineName, &t.Timestamp, &t.Input, &t.Output,
			&t.TotalElapsedMs, &t.TotalInputTokens, &t.TotalOutputTokens,
			&t.TotalToolCalls, &t.Status, &spansJSON,
		); err != nil {
			return nil, fmt.Errorf("scan trace: %w", err)
		}
		if err := json.Unmarshal([]byte(spansJSON), &t.Spans); err != nil {
			return nil, fmt.Errorf("unmarshal spans: %w", err)
		}
		traces = append(traces, t)
	}
	return traces, rows.Err()
}

func (s *SQLiteTraceStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM traces WHERE trace_id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete trace: %w", err)
	}
	return nil
}

func (s *SQLiteTraceStore) Summary(ctx context.Context) (MetricsSummary, error) {
	var m MetricsSummary
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(total_input_tokens), 0),
			COALESCE(SUM(total_output_tokens), 0),
			COALESCE(SUM(total_tool_calls), 0),
			COALESCE(AVG(total_elapsed_ms), 0)
		FROM traces`).Scan(
		&m.TotalTraces, &m.TotalInputTokens, &m.TotalOutputTokens,
		&m.TotalToolCalls, &m.AvgLatencyMs,
	)
	if err != nil {
		return m, fmt.Errorf("query summary: %w", err)
	}
	return m, nil
}

func (s *SQLiteTraceStore) Close() error {
	return s.db.Close()
}

// PipelineStore implementation

func (s *SQLitePipelineStore) Save(ctx context.Context, p PipelineInfo) error {
	nodes, err := json.Marshal(p.Nodes)
	if err != nil {
		return fmt.Errorf("marshal nodes: %w", err)
	}
	edges, err := json.Marshal(p.Edges)
	if err != nil {
		return fmt.Errorf("marshal edges: %w", err)
	}
	layout, err := json.Marshal(p.Layout)
	if err != nil {
		return fmt.Errorf("marshal layout: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO pipelines (id, name, description, nodes, edges, layout)
		VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Description, string(nodes), string(edges), string(layout),
	)
	if err != nil {
		return fmt.Errorf("insert pipeline: %w", err)
	}
	return nil
}

func (s *SQLitePipelineStore) Get(ctx context.Context, id string) (PipelineInfo, error) {
	var p PipelineInfo
	var nodesJSON, edgesJSON, layoutJSON string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, nodes, edges, layout
		FROM pipelines WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.Description, &nodesJSON, &edgesJSON, &layoutJSON,
	)
	if err == sql.ErrNoRows {
		return p, ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("query pipeline: %w", err)
	}

	if err := json.Unmarshal([]byte(nodesJSON), &p.Nodes); err != nil {
		return p, fmt.Errorf("unmarshal nodes: %w", err)
	}
	if err := json.Unmarshal([]byte(edgesJSON), &p.Edges); err != nil {
		return p, fmt.Errorf("unmarshal edges: %w", err)
	}
	if err := json.Unmarshal([]byte(layoutJSON), &p.Layout); err != nil {
		return p, fmt.Errorf("unmarshal layout: %w", err)
	}
	return p, nil
}

func (s *SQLitePipelineStore) List(ctx context.Context) ([]PipelineInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, nodes, edges, layout
		FROM pipelines ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("query pipelines: %w", err)
	}
	defer rows.Close()

	var pipelines []PipelineInfo
	for rows.Next() {
		var p PipelineInfo
		var nodesJSON, edgesJSON, layoutJSON string
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &nodesJSON, &edgesJSON, &layoutJSON); err != nil {
			return nil, fmt.Errorf("scan pipeline: %w", err)
		}
		if err := json.Unmarshal([]byte(nodesJSON), &p.Nodes); err != nil {
			return nil, fmt.Errorf("unmarshal nodes: %w", err)
		}
		if err := json.Unmarshal([]byte(edgesJSON), &p.Edges); err != nil {
			return nil, fmt.Errorf("unmarshal edges: %w", err)
		}
		if err := json.Unmarshal([]byte(layoutJSON), &p.Layout); err != nil {
			return nil, fmt.Errorf("unmarshal layout: %w", err)
		}
		pipelines = append(pipelines, p)
	}
	return pipelines, rows.Err()
}

func (s *SQLitePipelineStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pipelines WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}
	return nil
}

func (s *SQLitePipelineStore) Close() error {
	return s.db.Close()
}
