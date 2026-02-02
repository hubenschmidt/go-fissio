package llm

import "github.com/hubenschmidt/go-fissio/core"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LLMResponse struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason,omitempty"`
	Usage        Usage  `json:"usage,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponse struct {
	Content      string          `json:"content"`
	ToolCalls    []core.ToolCall `json:"tool_calls,omitempty"`
	FinishReason string          `json:"finish_reason,omitempty"`
	Usage        Usage           `json:"usage,omitempty"`
}

type StreamChunk struct {
	Content   string          `json:"content,omitempty"`
	ToolCalls []core.ToolCall `json:"tool_calls,omitempty"`
	Done      bool            `json:"done"`
	Error     error           `json:"error,omitempty"`
	Usage     *Usage          `json:"usage,omitempty"`
}

func (r *ChatResponse) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// EmbeddingResponse represents a single embedding result.
type EmbeddingResponse struct {
	Embedding  []float64 `json:"embedding"`
	TokenCount int       `json:"token_count"`
}
