package monitor

import (
	"sync"
	"time"
)

type MetricsCollector interface {
	Record(metrics NodeMetrics)
	Flush() PipelineMetrics
}

type InMemoryCollector struct {
	mu         sync.RWMutex
	pipelineID string
	metrics    map[string]NodeMetrics
	startTime  time.Time
}

func NewInMemoryCollector(pipelineID string) *InMemoryCollector {
	return &InMemoryCollector{
		pipelineID: pipelineID,
		metrics:    make(map[string]NodeMetrics),
		startTime:  time.Now(),
	}
}

func (c *InMemoryCollector) Record(metrics NodeMetrics) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics[metrics.NodeID] = metrics
}

func (c *InMemoryCollector) Flush() PipelineMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalTokens int
	var totalDuration time.Duration

	nodeMetrics := make(map[string]NodeMetrics, len(c.metrics))
	for k, v := range c.metrics {
		nodeMetrics[k] = v
		totalTokens += v.TokensIn + v.TokensOut
		totalDuration += v.Duration
	}

	return PipelineMetrics{
		PipelineID:    c.pipelineID,
		TotalTokens:   totalTokens,
		TotalDuration: totalDuration,
		NodeMetrics:   nodeMetrics,
		StartTime:     c.startTime,
		EndTime:       time.Now(),
	}
}

func (c *InMemoryCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = make(map[string]NodeMetrics)
	c.startTime = time.Now()
}

type NoOpCollector struct{}

func NewNoOpCollector() *NoOpCollector {
	return &NoOpCollector{}
}

func (c *NoOpCollector) Record(metrics NodeMetrics) {}

func (c *NoOpCollector) Flush() PipelineMetrics {
	return PipelineMetrics{}
}
