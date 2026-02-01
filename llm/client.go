package llm

import (
	"context"

	"github.com/hubenschmidt/go-fissio/core"
)

type Client interface {
	Chat(ctx context.Context, model string, system, user string) (*LLMResponse, error)
	ChatWithMessages(ctx context.Context, model string, system string, msgs []Message) (*ChatResponse, error)
	ChatWithTools(ctx context.Context, model string, system string, msgs []core.Message, tools []core.ToolSchema, pending []core.ToolResult) (*ChatResponse, error)
}

type StreamClient interface {
	Client
	ChatStream(ctx context.Context, model string, system, user string) (<-chan StreamChunk, error)
}

type ClientConfig struct {
	APIKey      string
	BaseURL     string
	Timeout     int
	MaxRetries  int
	DefaultModel string
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Timeout:    60,
		MaxRetries: 3,
	}
}
