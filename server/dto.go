package server

import "encoding/json"

type ModelInfo struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Model   string  `json:"model"`
	APIBase *string `json:"api_base,omitempty"`
}

type NodeInfo struct {
	ID       string   `json:"id"`
	NodeType string   `json:"node_type"`
	Model    *string  `json:"model,omitempty"`
	Prompt   *string  `json:"prompt,omitempty"`
	Tools    []string `json:"tools,omitempty"`
	X        *float64 `json:"x,omitempty"`
	Y        *float64 `json:"y,omitempty"`
}

type EdgeInfo struct {
	From     json.RawMessage `json:"from"`
	To       json.RawMessage `json:"to"`
	EdgeType *string         `json:"edge_type,omitempty"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type PipelineInfo struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Nodes       []NodeInfo          `json:"nodes"`
	Edges       []EdgeInfo          `json:"edges"`
	Layout      map[string]Position `json:"layout,omitempty"`
}

type InitResponse struct {
	Models    []ModelInfo    `json:"models"`
	Templates []PipelineInfo `json:"templates"`
	Configs   []PipelineInfo `json:"configs"`
}

type ToolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type SavePipelineRequest struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Nodes       []NodeInfo          `json:"nodes"`
	Edges       []EdgeInfo          `json:"edges"`
	Layout      map[string]Position `json:"layout,omitempty"`
}

type ChatRequest struct {
	Message      string           `json:"message"`
	ModelID      string           `json:"model_id,omitempty"`
	Pipeline     json.RawMessage  `json:"pipeline_config,omitempty"`
	SystemPrompt string           `json:"system_prompt,omitempty"`
	History      []HistoryMessage `json:"history,omitempty"`
}

type HistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	Content  string     `json:"content"`
	Metadata Metadata   `json:"metadata"`
}

type Metadata struct {
	InputTokens  int      `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	ElapsedMs    int64    `json:"elapsed_ms"`
	TokensPerSec *float64 `json:"tokens_per_sec,omitempty"`
}

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

type TraceListResponse struct {
	Traces []TraceInfo `json:"traces"`
}

type TraceDetailResponse struct {
	Trace TraceInfo  `json:"trace"`
	Spans []SpanInfo `json:"spans"`
}

type MetricsSummary struct {
	TotalTraces       int     `json:"total_traces"`
	TotalInputTokens  int     `json:"total_input_tokens"`
	TotalOutputTokens int     `json:"total_output_tokens"`
	TotalToolCalls    int     `json:"total_tool_calls"`
	AvgLatencyMs      float64 `json:"avg_latency_ms"`
}
