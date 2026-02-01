package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hubenschmidt/go-fissio/core"
)

type OpenAIClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func NewOpenAIClientWithConfig(cfg ClientConfig) *OpenAIClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIClient{
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
		client:  &http.Client{Timeout: time.Duration(cfg.Timeout) * time.Second},
	}
}

func (c *OpenAIClient) Chat(ctx context.Context, model string, system, user string) (*LLMResponse, error) {
	msgs := []core.Message{
		core.NewSystemMessage(system),
		core.NewUserMessage(user),
	}
	resp, err := c.ChatWithTools(ctx, model, "", msgs, nil, nil)
	if err != nil {
		return nil, err
	}
	return &LLMResponse{
		Content:      resp.Content,
		FinishReason: resp.FinishReason,
		Usage:        resp.Usage,
	}, nil
}

func (c *OpenAIClient) ChatWithMessages(ctx context.Context, model string, system string, msgs []Message) (*ChatResponse, error) {
	coreMsgs := make([]core.Message, len(msgs))
	for i, m := range msgs {
		coreMsgs[i] = core.Message{Role: core.MessageRole(m.Role), Content: m.Content}
	}
	return c.ChatWithTools(ctx, model, system, coreMsgs, nil, nil)
}

func (c *OpenAIClient) ChatWithTools(ctx context.Context, model string, system string, msgs []core.Message, tools []core.ToolSchema, pending []core.ToolResult) (*ChatResponse, error) {
	messages := c.buildMessages(system, msgs, pending)

	reqBody := map[string]any{
		"model":    model,
		"messages": messages,
	}

	if len(tools) > 0 {
		reqBody["tools"] = c.buildTools(tools)
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return c.parseResponse(result), nil
}

func (c *OpenAIClient) buildMessages(system string, msgs []core.Message, pending []core.ToolResult) []map[string]any {
	messages := make([]map[string]any, 0, len(msgs)+len(pending)+1)

	if system != "" {
		messages = append(messages, map[string]any{
			"role":    "system",
			"content": system,
		})
	}

	for _, m := range msgs {
		msg := map[string]any{
			"role":    string(m.Role),
			"content": m.Content,
		}
		if m.ToolCallID != "" {
			msg["tool_call_id"] = m.ToolCallID
		}
		messages = append(messages, msg)
	}

	for _, p := range pending {
		messages = append(messages, map[string]any{
			"role":         "tool",
			"content":      p.Content,
			"tool_call_id": p.ToolCallID,
		})
	}

	return messages
}

func (c *OpenAIClient) buildTools(tools []core.ToolSchema) []map[string]any {
	result := make([]map[string]any, len(tools))
	for i, t := range tools {
		result[i] = map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  json.RawMessage(t.Parameters),
			},
		}
	}
	return result
}

func (c *OpenAIClient) parseResponse(resp openAIResponse) *ChatResponse {
	if len(resp.Choices) == 0 {
		return &ChatResponse{}
	}

	choice := resp.Choices[0]
	result := &ChatResponse{
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	for _, tc := range choice.Message.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, core.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: json.RawMessage(tc.Function.Arguments),
		})
	}

	return result
}

type openAIResponse struct {
	Choices []openAIChoice `json:"choices"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIChoice struct {
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func (c *OpenAIClient) ChatStreamWithMessages(ctx context.Context, model string, system string, msgs []Message) (<-chan StreamChunk, error) {
	coreMsgs := make([]core.Message, len(msgs))
	for i, m := range msgs {
		coreMsgs[i] = core.Message{Role: core.MessageRole(m.Role), Content: m.Content}
	}

	messages := c.buildMessages(system, coreMsgs, nil)

	reqBody := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	ch := make(chan StreamChunk)
	go c.readStream(resp, ch)
	return ch, nil
}

func (c *OpenAIClient) readStream(resp *http.Response, ch chan<- StreamChunk) {
	defer resp.Body.Close()
	defer close(ch)

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				ch <- StreamChunk{Error: err, Done: true}
			} else {
				ch <- StreamChunk{Done: true}
			}
			return
		}

		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			if line == "data: [DONE]" {
				ch <- StreamChunk{Done: true}
				return
			}
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			ch <- StreamChunk{Content: chunk.Choices[0].Delta.Content}
		}
	}
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}
