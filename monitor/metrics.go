package monitor

import "time"

type NodeMetrics struct {
	NodeID    string        `json:"node_id"`
	TokensIn  int           `json:"tokens_in"`
	TokensOut int           `json:"tokens_out"`
	Duration  time.Duration `json:"duration"`
	Success   bool          `json:"success"`
	Error     string        `json:"error,omitempty"`
}

type PipelineMetrics struct {
	PipelineID   string                 `json:"pipeline_id"`
	TotalTokens  int                    `json:"total_tokens"`
	TotalDuration time.Duration         `json:"total_duration"`
	NodeMetrics  map[string]NodeMetrics `json:"node_metrics"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
}

type ObserveConfig struct {
	EnableMetrics bool `json:"enable_metrics"`
	EnableTracing bool `json:"enable_tracing"`
	SampleRate    float64 `json:"sample_rate"`
}

func DefaultObserveConfig() ObserveConfig {
	return ObserveConfig{
		EnableMetrics: true,
		EnableTracing: false,
		SampleRate:    1.0,
	}
}
