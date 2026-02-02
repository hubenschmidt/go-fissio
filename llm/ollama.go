package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type OllamaTagsResponse struct {
	Models []OllamaModelInfo `json:"models"`
}

type OllamaModelInfo struct {
	Name string `json:"name"`
}

type DiscoveredModel struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Model   string  `json:"model"`
	APIBase *string `json:"api_base,omitempty"`
}

// DiscoverOllamaModels queries an Ollama instance for available models.
func DiscoverOllamaModels(ollamaHost string) ([]DiscoveredModel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := strings.TrimSuffix(ollamaHost, "/")
	// Handle both /v1 suffix and bare host
	host = strings.TrimSuffix(host, "/v1")
	url := fmt.Sprintf("%s/api/tags", host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama discovery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var tags OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	apiBase := fmt.Sprintf("%s/v1", host)
	models := make([]DiscoveredModel, len(tags.Models))
	for i, m := range tags.Models {
		models[i] = DiscoveredModel{
			ID:      fmt.Sprintf("ollama-%s", slugify(m.Name)),
			Name:    formatDisplayName(m.Name),
			Model:   m.Name,
			APIBase: &apiBase,
		}
	}

	return models, nil
}

func slugify(name string) string {
	// Replace colons and special chars with dashes
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	return strings.ToLower(re.ReplaceAllString(name, "-"))
}

func formatDisplayName(name string) string {
	// "llama3.2:latest" -> "Llama3.2 (Ollama)"
	parts := strings.Split(name, ":")
	base := parts[0]
	// Capitalize first letter
	if len(base) > 0 {
		base = strings.ToUpper(base[:1]) + base[1:]
	}
	return fmt.Sprintf("%s (Ollama)", base)
}

// OllamaEmbedClient handles Ollama-native embedding API.
type OllamaEmbedClient struct {
	baseURL string
	client  *http.Client
}

// NewOllamaEmbedClient creates a client for Ollama's native embedding API.
func NewOllamaEmbedClient(baseURL string) *OllamaEmbedClient {
	host := strings.TrimSuffix(baseURL, "/")
	host = strings.TrimSuffix(host, "/v1")
	return &OllamaEmbedClient{
		baseURL: host,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

// Embed generates an embedding for a single input using Ollama's native API.
func (c *OllamaEmbedClient) Embed(ctx context.Context, model, input string) (*EmbeddingResponse, error) {
	results, err := c.EmbedBatch(ctx, model, []string{input})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return &results[0], nil
}

// EmbedBatch generates embeddings for multiple inputs using Ollama's native API.
func (c *OllamaEmbedClient) EmbedBatch(ctx context.Context, model string, inputs []string) ([]EmbeddingResponse, error) {
	results := make([]EmbeddingResponse, 0, len(inputs))

	// Ollama's /api/embed endpoint processes one input at a time
	for _, input := range inputs {
		reqBody := map[string]any{
			"model": model,
			"input": input,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embed", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(respBody))
		}

		var result ollamaEmbedResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		resp.Body.Close()

		if len(result.Embeddings) == 0 {
			return nil, fmt.Errorf("no embeddings in response")
		}

		results = append(results, EmbeddingResponse{
			Embedding:  result.Embeddings[0],
			TokenCount: 0, // Ollama doesn't report token counts
		})
	}

	return results, nil
}

type ollamaEmbedResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}
