package core

type ModelConfig struct {
	Name        string  `json:"name"`
	Provider    string  `json:"provider,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

func DefaultModelConfig(name string) ModelConfig {
	return ModelConfig{
		Name:        name,
		Temperature: 0.7,
		MaxTokens:   4096,
	}
}

func (m ModelConfig) WithTemperature(t float64) ModelConfig {
	m.Temperature = t
	return m
}

func (m ModelConfig) WithMaxTokens(t int) ModelConfig {
	m.MaxTokens = t
	return m
}

func (m ModelConfig) WithProvider(p string) ModelConfig {
	m.Provider = p
	return m
}
