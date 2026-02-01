package core

import "encoding/json"

type ToolCall struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error,omitempty"`
}

type ToolSchema struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

func NewToolResult(toolCallID, content string) ToolResult {
	return ToolResult{ToolCallID: toolCallID, Content: content}
}

func NewToolError(toolCallID, errMsg string) ToolResult {
	return ToolResult{ToolCallID: toolCallID, Content: errMsg, IsError: true}
}
