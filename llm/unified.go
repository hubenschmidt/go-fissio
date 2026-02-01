package llm

import (
	"context"
	"strings"

	"github.com/hubenschmidt/go-fissio/core"
)

type UnifiedClient struct {
	openai    *OpenAIClient
	anthropic *AnthropicClient
	ollama    *OpenAIClient
}

type UnifiedConfig struct {
	OpenAIKey    string
	AnthropicKey string
	OllamaURL    string
}

func NewUnifiedClient(cfg UnifiedConfig) *UnifiedClient {
	u := &UnifiedClient{}

	if cfg.OpenAIKey != "" {
		u.openai = NewOpenAIClient(cfg.OpenAIKey)
	}

	if cfg.AnthropicKey != "" {
		u.anthropic = NewAnthropicClient(cfg.AnthropicKey)
	}

	if cfg.OllamaURL != "" {
		u.ollama = NewOpenAIClientWithConfig(ClientConfig{
			BaseURL: cfg.OllamaURL,
		})
	}

	return u
}

func (u *UnifiedClient) Chat(ctx context.Context, model string, system, user string) (*LLMResponse, error) {
	client, resolvedModel := u.resolveClient(model)
	return client.Chat(ctx, resolvedModel, system, user)
}

func (u *UnifiedClient) ChatWithMessages(ctx context.Context, model string, system string, msgs []Message) (*ChatResponse, error) {
	client, resolvedModel := u.resolveClient(model)
	return client.ChatWithMessages(ctx, resolvedModel, system, msgs)
}

func (u *UnifiedClient) ChatWithTools(ctx context.Context, model string, system string, msgs []core.Message, tools []core.ToolSchema, pending []core.ToolResult) (*ChatResponse, error) {
	client, resolvedModel := u.resolveClient(model)
	return client.ChatWithTools(ctx, resolvedModel, system, msgs, tools, pending)
}

func (u *UnifiedClient) resolveClient(model string) (Client, string) {
	prefixMap := map[string]struct {
		client Client
		strip  bool
	}{
		"claude-":  {u.anthropic, false},
		"gpt-":     {u.openai, false},
		"o1-":      {u.openai, false},
		"ollama/":  {u.ollama, true},
	}

	for prefix, cfg := range prefixMap {
		if !strings.HasPrefix(model, prefix) {
			continue
		}
		if cfg.client == nil {
			continue
		}
		resolvedModel := model
		if cfg.strip {
			resolvedModel = strings.TrimPrefix(model, prefix)
		}
		return cfg.client, resolvedModel
	}

	if u.openai != nil {
		return u.openai, model
	}
	if u.anthropic != nil {
		return u.anthropic, model
	}
	if u.ollama != nil {
		return u.ollama, model
	}

	return nil, model
}

func (u *UnifiedClient) HasOpenAI() bool {
	return u.openai != nil
}

func (u *UnifiedClient) HasAnthropic() bool {
	return u.anthropic != nil
}

func (u *UnifiedClient) HasOllama() bool {
	return u.ollama != nil
}
