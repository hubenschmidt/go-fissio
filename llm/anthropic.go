package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hubenschmidt/go-fissio/core"
)

type AnthropicClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
	version string
}

func NewAnthropicClient(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		apiKey:  apiKey,
		baseURL: "https://api.anthropic.com/v1",
		client:  &http.Client{Timeout: 60 * time.Second},
		version: "2023-06-01",
	}
}

func NewAnthropicClientWithConfig(cfg ClientConfig) *AnthropicClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	return &AnthropicClient{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second},
		version: "2023-06-01",
	}
}

func (c *AnthropicClient) Chat(ctx context.Context, model string, system, user string) (*LLMResponse, error) {
	msgs := []core.Message{core.NewUserMessage(user)}
	resp, err := c.ChatWithTools(ctx, model, system, msgs, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LLMResponse{
		Content:      resp.Content,
		FinishReason: resp.FinishReason,
		Usage:        resp.Usage,
	}, nil
}

func (c *AnthropicClient) ChatWithMessages(ctx context.Context, model string, system string, msgs []Message) (*ChatResponse, error) {
	coreMsgs := make([]core.Message, len(msgs))
	for i, m := range msgs {
		coreMsgs[i] = core.Message{Role: core.MessageRole(m.Role), Content: m.Content}
	}
	return c.ChatWithTools(ctx, model, system, coreMsgs, nil, nil)
}

func (c *AnthropicClient) ChatWithTools(ctx context.Context, model string, system string, msgs []core.Message, tools []core.ToolSchema, pending []core.ToolResult) (*ChatResponse, error) {
	reqBody := map[string]any{
		"model":      model,
		"max_tokens": 4096,
		"messages":   c.buildMessages(msgs, pending),
	}

	if system != "" {
		reqBody["system"] = system
	}

	if len(tools) > 0 {
		reqBody["tools"] = c.buildTools(tools)
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", c.version)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.parseResponse(result), nil
}

func (c *AnthropicClient) buildMessages(msgs []core.Message, pending []core.ToolResult) []map[string]any {
	messages := make([]map[string]any, 0, len(msgs)+len(pending))

	nonSystemMsgs := filterMessages(msgs, func(m core.Message) bool {
		return m.Role != core.RoleSystem
	})

	for _, m := range nonSystemMsgs {
		messages = append(messages, c.convertMessage(m))
	}

	for _, p := range pending {
		messages = append(messages, c.toolResultMessage(p.ToolCallID, p.Content))
	}

	return messages
}

func (c *AnthropicClient) convertMessage(m core.Message) map[string]any {
	if m.Role == core.RoleTool {
		return c.toolResultMessage(m.ToolCallID, m.Content)
	}
	return map[string]any{
		"role":    string(m.Role),
		"content": m.Content,
	}
}

func (c *AnthropicClient) toolResultMessage(toolCallID, content string) map[string]any {
	return map[string]any{
		"role": "user",
		"content": []map[string]any{{
			"type":        "tool_result",
			"tool_use_id": toolCallID,
			"content":     content,
		}},
	}
}

func filterMessages(msgs []core.Message, predicate func(core.Message) bool) []core.Message {
	result := make([]core.Message, 0, len(msgs))
	for _, m := range msgs {
		if predicate(m) {
			result = append(result, m)
		}
	}
	return result
}

func (c *AnthropicClient) buildTools(tools []core.ToolSchema) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, t := range tools {
		result[i] = map[string]any{
			"name":         t.Name,
			"description":  t.Description,
			"input_schema": json.RawMessage(t.Parameters),
		}
	}
	return result
}

func (c *AnthropicClient) parseResponse(resp anthropicResponse) *ChatResponse {
	result := &ChatResponse{
		FinishReason: resp.StopReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	blockHandlers := map[string]func(*ChatResponse, anthropicBlock){
		"text":     c.handleTextBlock,
		"tool_use": c.handleToolUseBlock,
	}

	for _, block := range resp.Content {
		if handler, ok := blockHandlers[block.Type]; ok {
			handler(result, block)
		}
	}

	return result
}

func (c *AnthropicClient) handleTextBlock(result *ChatResponse, block anthropicBlock) {
	result.Content += block.Text
}

func (c *AnthropicClient) handleToolUseBlock(result *ChatResponse, block anthropicBlock) {
	inputBytes, _ := json.Marshal(block.Input)
	result.ToolCalls = append(result.ToolCalls, core.ToolCall{
		ID:        block.ID,
		Name:      block.Name,
		Arguments: inputBytes,
	})
}

type anthropicResponse struct {
	Content    []anthropicBlock `json:"content"`
	StopReason string           `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

type anthropicBlock struct {
	Type  string         `json:"type"`
	Text  string         `json:"text,omitempty"`
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`
}
