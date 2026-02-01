package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hubenschmidt/go-fissio/server/store/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresTraceStore implements TraceStore using PostgreSQL
type PostgresTraceStore struct {
	db *sql.DB
}

// PostgresPipelineStore implements PipelineStore using PostgreSQL
type PostgresPipelineStore struct {
	db *sql.DB
}

// NewPostgresStores creates PostgreSQL-backed trace and pipeline stores
func NewPostgresStores(dsn string) (TraceStore, PipelineStore, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := runPostgresMigrations(db); err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	return &PostgresTraceStore{db: db}, &PostgresPipelineStore{db: db}, nil
}

func runPostgresMigrations(db *sql.DB) error {
	data, err := migrations.Postgres.ReadFile("postgres/001_init.sql")
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

func (s *PostgresTraceStore) Add(ctx context.Context, t TraceInfo) error {
	spans, err := json.Marshal(t.Spans)
	if err != nil {
		return fmt.Errorf("marshal spans: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO traces (
			trace_id, pipeline_id, pipeline_name, timestamp, input, output,
			total_elapsed_ms, total_input_tokens, total_output_tokens,
			total_tool_calls, status, spans
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (trace_id) DO UPDATE SET
			pipeline_id = EXCLUDED.pipeline_id,
			pipeline_name = EXCLUDED.pipeline_name,
			timestamp = EXCLUDED.timestamp,
			input = EXCLUDED.input,
			output = EXCLUDED.output,
			total_elapsed_ms = EXCLUDED.total_elapsed_ms,
			total_input_tokens = EXCLUDED.total_input_tokens,
			total_output_tokens = EXCLUDED.total_output_tokens,
			total_tool_calls = EXCLUDED.total_tool_calls,
			status = EXCLUDED.status,
			spans = EXCLUDED.spans`,
		t.TraceID, t.PipelineID, t.PipelineName, t.Timestamp, t.Input, t.Output,
		t.TotalElapsedMs, t.TotalInputTokens, t.TotalOutputTokens,
		t.TotalToolCalls, t.Status, spans,
	)
	if err != nil {
		return fmt.Errorf("insert trace: %w", err)
	}
	return nil
}

func (s *PostgresTraceStore) Get(ctx context.Context, id string) (TraceInfo, error) {
	var t TraceInfo
	var spansJSON []byte

	err := s.db.QueryRowContext(ctx, `
		SELECT trace_id, pipeline_id, pipeline_name, timestamp, input, output,
			   total_elapsed_ms, total_input_tokens, total_output_tokens,
			   total_tool_calls, status, spans
		FROM traces WHERE trace_id = $1`, id).Scan(
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

	if err := json.Unmarshal(spansJSON, &t.Spans); err != nil {
		return t, fmt.Errorf("unmarshal spans: %w", err)
	}
	return t, nil
}

func (s *PostgresTraceStore) List(ctx context.Context) ([]TraceInfo, error) {
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
		var spansJSON []byte
		if err := rows.Scan(
			&t.TraceID, &t.PipelineID, &t.PipelineName, &t.Timestamp, &t.Input, &t.Output,
			&t.TotalElapsedMs, &t.TotalInputTokens, &t.TotalOutputTokens,
			&t.TotalToolCalls, &t.Status, &spansJSON,
		); err != nil {
			return nil, fmt.Errorf("scan trace: %w", err)
		}
		if err := json.Unmarshal(spansJSON, &t.Spans); err != nil {
			return nil, fmt.Errorf("unmarshal spans: %w", err)
		}
		traces = append(traces, t)
	}
	return traces, rows.Err()
}

func (s *PostgresTraceStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM traces WHERE trace_id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete trace: %w", err)
	}
	return nil
}

func (s *PostgresTraceStore) Summary(ctx context.Context) (MetricsSummary, error) {
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

func (s *PostgresTraceStore) Close() error {
	return s.db.Close()
}

// PipelineStore implementation

func (s *PostgresPipelineStore) Save(ctx context.Context, p PipelineInfo) error {
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
		INSERT INTO pipelines (id, name, description, nodes, edges, layout)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			nodes = EXCLUDED.nodes,
			edges = EXCLUDED.edges,
			layout = EXCLUDED.layout`,
		p.ID, p.Name, p.Description, nodes, edges, layout,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline: %w", err)
	}
	return nil
}

func (s *PostgresPipelineStore) Get(ctx context.Context, id string) (PipelineInfo, error) {
	var p PipelineInfo
	var nodesJSON, edgesJSON, layoutJSON []byte

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, nodes, edges, layout
		FROM pipelines WHERE id = $1`, id).Scan(
		&p.ID, &p.Name, &p.Description, &nodesJSON, &edgesJSON, &layoutJSON,
	)
	if err == sql.ErrNoRows {
		return p, ErrNotFound
	}
	if err != nil {
		return p, fmt.Errorf("query pipeline: %w", err)
	}

	if err := json.Unmarshal(nodesJSON, &p.Nodes); err != nil {
		return p, fmt.Errorf("unmarshal nodes: %w", err)
	}
	if err := json.Unmarshal(edgesJSON, &p.Edges); err != nil {
		return p, fmt.Errorf("unmarshal edges: %w", err)
	}
	if err := json.Unmarshal(layoutJSON, &p.Layout); err != nil {
		return p, fmt.Errorf("unmarshal layout: %w", err)
	}
	return p, nil
}

func (s *PostgresPipelineStore) List(ctx context.Context) ([]PipelineInfo, error) {
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
		var nodesJSON, edgesJSON, layoutJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &nodesJSON, &edgesJSON, &layoutJSON); err != nil {
			return nil, fmt.Errorf("scan pipeline: %w", err)
		}
		if err := json.Unmarshal(nodesJSON, &p.Nodes); err != nil {
			return nil, fmt.Errorf("unmarshal nodes: %w", err)
		}
		if err := json.Unmarshal(edgesJSON, &p.Edges); err != nil {
			return nil, fmt.Errorf("unmarshal edges: %w", err)
		}
		if err := json.Unmarshal(layoutJSON, &p.Layout); err != nil {
			return nil, fmt.Errorf("unmarshal layout: %w", err)
		}
		pipelines = append(pipelines, p)
	}
	return pipelines, rows.Err()
}

func (s *PostgresPipelineStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pipelines WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete pipeline: %w", err)
	}
	return nil
}

func (s *PostgresPipelineStore) Close() error {
	return s.db.Close()
}
