package llm

import (
	"context"
	"encoding/json"
	"fmt"
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
