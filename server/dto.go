package server

import (
	"encoding/json"

	"github.com/hubenschmidt/go-fissio/server/store"
)

type ModelInfo struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Model   string  `json:"model"`
	APIBase *string `json:"api_base,omitempty"`
}

// Re-export types from store package
type (
	NodeInfo       = store.NodeInfo
	EdgeInfo       = store.EdgeInfo
	Position       = store.Position
	PipelineInfo   = store.PipelineInfo
	TraceInfo      = store.TraceInfo
	SpanInfo       = store.SpanInfo
	MetricsSummary = store.MetricsSummary
)

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

type TraceListResponse struct {
	Traces []TraceInfo `json:"traces"`
}

type TraceDetailResponse struct {
	Trace TraceInfo  `json:"trace"`
	Spans []SpanInfo `json:"spans"`
}
