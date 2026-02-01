package server

import "sync"

type PipelineStore struct {
	mu        sync.RWMutex
	pipelines map[string]PipelineInfo
}

func newPipelineStore() *PipelineStore {
	return &PipelineStore{
		pipelines: make(map[string]PipelineInfo),
	}
}

func (s *PipelineStore) List() []PipelineInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]PipelineInfo, 0, len(s.pipelines))
	for _, p := range s.pipelines {
		result = append(result, p)
	}
	return result
}

func (s *PipelineStore) Save(p PipelineInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pipelines[p.ID] = p
}

func (s *PipelineStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pipelines, id)
}

type TraceStore struct {
	mu     sync.RWMutex
	traces map[string]TraceInfo
}

func newTraceStore() *TraceStore {
	return &TraceStore{
		traces: make(map[string]TraceInfo),
	}
}

func (s *TraceStore) Add(t TraceInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.traces[t.TraceID] = t
}

func (s *TraceStore) List() []TraceInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]TraceInfo, 0, len(s.traces))
	for _, t := range s.traces {
		result = append(result, t)
	}
	return result
}

func (s *TraceStore) Get(id string) (TraceInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.traces[id]
	return t, ok
}

func (s *TraceStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.traces, id)
}

func (s *TraceStore) Summary() MetricsSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.traces) == 0 {
		return MetricsSummary{}
	}

	var totalLatency int64
	var totalIn, totalOut, totalTools int
	for _, t := range s.traces {
		totalLatency += t.TotalElapsedMs
		totalIn += t.TotalInputTokens
		totalOut += t.TotalOutputTokens
		totalTools += t.TotalToolCalls
	}

	return MetricsSummary{
		TotalTraces:       len(s.traces),
		TotalInputTokens:  totalIn,
		TotalOutputTokens: totalOut,
		TotalToolCalls:    totalTools,
		AvgLatencyMs:      float64(totalLatency) / float64(len(s.traces)),
	}
}
