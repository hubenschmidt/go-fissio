package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type FetchURL struct {
	client *http.Client
}

type fetchURLArgs struct {
	URL     string `json:"url"`
	Timeout int    `json:"timeout,omitempty"`
}

func NewFetchURL() *FetchURL {
	return &FetchURL{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *FetchURL) Name() string {
	return "fetch_url"
}

func (f *FetchURL) Description() string {
	return "Fetches content from a URL and returns the response body"
}

func (f *FetchURL) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"url": {
				"type": "string",
				"description": "The URL to fetch"
			},
			"timeout": {
				"type": "integer",
				"description": "Timeout in seconds (default: 30)"
			}
		},
		"required": ["url"]
	}`)
}

func (f *FetchURL) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params fetchURLArgs
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	timeout := 30 * time.Second
	if params.Timeout > 0 {
		timeout = time.Duration(params.Timeout) * time.Second
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return string(body), nil
}

func init() {
	Register(NewFetchURL())
}
