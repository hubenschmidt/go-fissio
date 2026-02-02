package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/hubenschmidt/go-fissio/core"
)

type UnifiedClient struct {
	openai      *OpenAIClient
	anthropic   *AnthropicClient
	ollama      *OpenAIClient
	ollamaEmbed *OllamaEmbedClient
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
		u.ollamaEmbed = NewOllamaEmbedClient(cfg.OllamaURL)
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

func (u *UnifiedClient) ChatStreamWithMessages(ctx context.Context, model string, system string, msgs []Message) (<-chan StreamChunk, error) {
	client, resolvedModel := u.resolveClient(model)
	if sc, ok := client.(*OpenAIClient); ok {
		return sc.ChatStreamWithMessages(ctx, resolvedModel, system, msgs)
	}
	// Fallback: non-streaming response wrapped in channel
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		resp, err := client.ChatWithMessages(ctx, resolvedModel, system, msgs)
		if err != nil {
			ch <- StreamChunk{Error: err, Done: true}
			return
		}
		ch <- StreamChunk{Content: resp.Content}
		ch <- StreamChunk{Done: true, Usage: &resp.Usage}
	}()
	return ch, nil
}

func (u *UnifiedClient) ChatWithTools(ctx context.Context, model string, system string, msgs []core.Message, tools []core.ToolSchema, pending []core.ToolResult) (*ChatResponse, error) {
	client, resolvedModel := u.resolveClient(model)
	return client.ChatWithTools(ctx, resolvedModel, system, msgs, tools, pending)
}

func (u *UnifiedClient) resolveClient(model string) (Client, string) {
	prefixes := []struct {
		prefix string
		client Client
		strip  bool
	}{
		{"claude-", u.anthropic, false},
		{"gpt-", u.openai, false},
		{"o1-", u.openai, false},
		{"ollama/", u.ollama, true},
	}

	for _, p := range prefixes {
		if strings.HasPrefix(model, p.prefix) && p.client != nil {
			resolvedModel := model
			if p.strip {
				resolvedModel = strings.TrimPrefix(model, p.prefix)
			}
			return p.client, resolvedModel
		}
	}

	return u.defaultClient(), model
}

func (u *UnifiedClient) defaultClient() Client {
	clients := []Client{u.openai, u.anthropic, u.ollama}
	for _, c := range clients {
		if c != nil {
			return c
		}
	}
	return nil
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

// Embed generates an embedding for a single input.
func (u *UnifiedClient) Embed(ctx context.Context, model, input string) (*EmbeddingResponse, error) {
	client, resolvedModel := u.resolveEmbeddingClient(model)
	if client == nil {
		return nil, fmt.Errorf("no embedding client available for model: %s", model)
	}
	return client.Embed(ctx, resolvedModel, input)
}

// EmbedBatch generates embeddings for multiple inputs.
func (u *UnifiedClient) EmbedBatch(ctx context.Context, model string, inputs []string) ([]EmbeddingResponse, error) {
	client, resolvedModel := u.resolveEmbeddingClient(model)
	if client == nil {
		return nil, fmt.Errorf("no embedding client available for model: %s", model)
	}
	return client.EmbedBatch(ctx, resolvedModel, inputs)
}

func (u *UnifiedClient) resolveEmbeddingClient(model string) (EmbeddingClient, string) {
	// Ollama embedding models
	if strings.HasPrefix(model, "ollama/") {
		if u.ollamaEmbed == nil {
			return nil, model
		}
		return u.ollamaEmbed, strings.TrimPrefix(model, "ollama/")
	}

	// OpenAI embedding models (text-embedding-3-small, text-embedding-3-large, etc.)
	if strings.HasPrefix(model, "text-embedding-") {
		if u.openai == nil {
			return nil, model
		}
		return u.openai, model
	}

	// Default to OpenAI for unknown embedding models
	if u.openai != nil {
		return u.openai, model
	}

	// Fall back to Ollama if OpenAI not available
	if u.ollamaEmbed != nil {
		return u.ollamaEmbed, model
	}

	return nil, model
}
