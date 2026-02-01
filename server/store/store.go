package store

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrNotFound is returned when an entity is not found
var ErrNotFound = errors.New("not found")

// TraceInfo represents a recorded trace
type TraceInfo struct {
	TraceID           string     `json:"trace_id"`
	PipelineID        string     `json:"pipeline_id"`
	PipelineName      string     `json:"pipeline_name"`
	Timestamp         int64      `json:"timestamp"`
	Input             string     `json:"input"`
	Output            string     `json:"output"`
	TotalElapsedMs    int64      `json:"total_elapsed_ms"`
	TotalInputTokens  int        `json:"total_input_tokens"`
	TotalOutputTokens int        `json:"total_output_tokens"`
	TotalToolCalls    int        `json:"total_tool_calls"`
	Status            string     `json:"status"`
	Spans             []SpanInfo `json:"spans,omitempty"`
}

// SpanInfo represents a span within a trace
type SpanInfo struct {
	SpanID         string `json:"span_id"`
	TraceID        string `json:"trace_id"`
	NodeID         string `json:"node_id"`
	NodeType       string `json:"node_type"`
	StartTime      int64  `json:"start_time"`
	EndTime        int64  `json:"end_time"`
	Input          string `json:"input"`
	Output         string `json:"output"`
	InputTokens    int    `json:"input_tokens"`
	OutputTokens   int    `json:"output_tokens"`
	ToolCallCount  int    `json:"tool_call_count"`
	IterationCount int    `json:"iteration_count"`
}

// MetricsSummary contains aggregated metrics
type MetricsSummary struct {
	TotalTraces       int     `json:"total_traces"`
	TotalInputTokens  int     `json:"total_input_tokens"`
	TotalOutputTokens int     `json:"total_output_tokens"`
	TotalToolCalls    int     `json:"total_tool_calls"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
}

// NodeInfo represents a node in a pipeline
type NodeInfo struct {
	ID       string   `json:"id"`
	NodeType string   `json:"node_type"`
	Model    *string  `json:"model,omitempty"`
	Prompt   *string  `json:"prompt,omitempty"`
	Tools    []string `json:"tools,omitempty"`
	X        *float64 `json:"x,omitempty"`
	Y        *float64 `json:"y,omitempty"`
}

// EdgeInfo represents an edge in a pipeline
type EdgeInfo struct {
	From     json.RawMessage `json:"from"`
	To       json.RawMessage `json:"to"`
	EdgeType *string         `json:"edge_type,omitempty"`
}

// Position represents a 2D position
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// PipelineInfo represents a saved pipeline configuration
type PipelineInfo struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Nodes       []NodeInfo          `json:"nodes"`
	Edges       []EdgeInfo          `json:"edges"`
	Layout      map[string]Position `json:"layout,omitempty"`
}

// TraceStore defines the interface for trace persistence
type TraceStore interface {
	Add(ctx context.Context, t TraceInfo) error
	Get(ctx context.Context, id string) (TraceInfo, error)
	List(ctx context.Context) ([]TraceInfo, error)
	Delete(ctx context.Context, id string) error
	Summary(ctx context.Context) (MetricsSummary, error)
	Close() error
}

// PipelineStore defines the interface for pipeline persistence
type PipelineStore interface {
	Save(ctx context.Context, p PipelineInfo) error
	Get(ctx context.Context, id string) (PipelineInfo, error)
	List(ctx context.Context) ([]PipelineInfo, error)
	Delete(ctx context.Context, id string) error
	Close() error
}
