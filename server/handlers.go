package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hubenschmidt/go-fissio/config"
	"github.com/hubenschmidt/go-fissio/core"
	"github.com/hubenschmidt/go-fissio/engine"
	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/tools"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func (s *Server) handleInit(w http.ResponseWriter, r *http.Request) {
	resp := InitResponse{
		Models:    s.models,
		Templates: s.templates,
		Configs:   s.pipelines.List(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleTools(w http.ResponseWriter, r *http.Request) {
	names := s.registry.List()
	result := make([]ToolInfo, 0, len(names))

	for _, name := range names {
		t, ok := s.registry.Get(name)
		if !ok {
			continue
		}
		result = append(result, ToolInfo{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Handle direct chat (no pipeline)
	if len(req.Pipeline) == 0 || string(req.Pipeline) == "null" {
		s.handleDirectChat(w, r, req, flusher)
		return
	}

	var rp runtimePipeline
	if err := json.Unmarshal(req.Pipeline, &rp); err != nil {
		writeSSE(w, flusher, "stream", map[string]any{"content": "Error: invalid pipeline config"})
		writeSSE(w, flusher, "end", nil)
		return
	}

	pipelineCfg := buildPipeline(rp)
	resolver := engine.NewModelResolver(core.DefaultModelConfig("gpt-4"))
	eng := engine.NewEngine(pipelineCfg, engine.EngineConfig{
		Client:   s.client,
		Registry: s.registry,
		Resolver: resolver,
	})

	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	result, err := eng.Run(ctx, req.Message)
	elapsed := time.Since(start)

	if err != nil {
		writeSSE(w, flusher, "stream", map[string]any{"content": "Error: " + err.Error()})
		writeSSE(w, flusher, "end", nil)
		return
	}

	var totalIn, totalOut int
	for _, out := range result.Outputs {
		totalIn += out.TokensIn
		totalOut += out.TokensOut
	}

	writeSSE(w, flusher, "stream", map[string]any{"content": result.Content})
	writeSSE(w, flusher, "end", map[string]any{
		"metadata": Metadata{
			InputTokens:  totalIn,
			OutputTokens: totalOut,
			ElapsedMs:    elapsed.Milliseconds(),
		},
	})

	// Record trace
	s.traces.Add(TraceInfo{
		TraceID:           fmt.Sprintf("trace_%d", time.Now().UnixNano()),
		PipelineID:        rp.Nodes[0].ID,
		PipelineName:      "Pipeline",
		Timestamp:         start.UnixMilli(),
		Input:             req.Message,
		Output:            result.Content,
		TotalElapsedMs:    elapsed.Milliseconds(),
		TotalInputTokens:  totalIn,
		TotalOutputTokens: totalOut,
		TotalToolCalls:    0,
		Status:            "success",
	})
}

func (s *Server) handleDirectChat(w http.ResponseWriter, r *http.Request, req ChatRequest, flusher http.Flusher) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	// Use provided system prompt or default
	systemPrompt := "You are a helpful assistant."
	if req.SystemPrompt != "" {
		systemPrompt = req.SystemPrompt
	}

	// Build messages with history if provided
	messages := make([]llm.Message, 0, len(req.History)+1)
	for _, h := range req.History {
		messages = append(messages, llm.Message{Role: h.Role, Content: h.Content})
	}
	messages = append(messages, llm.Message{Role: "user", Content: req.Message})
	resp, err := s.client.ChatWithMessages(ctx, "gpt-4", systemPrompt, messages)
	elapsed := time.Since(start)

	if err != nil {
		writeSSE(w, flusher, "stream", map[string]any{"content": "Error: " + err.Error()})
		writeSSE(w, flusher, "end", nil)
		s.traces.Add(TraceInfo{
			TraceID:        fmt.Sprintf("trace_%d", time.Now().UnixNano()),
			PipelineID:     "direct",
			PipelineName:   "Direct Chat",
			Timestamp:      start.UnixMilli(),
			Input:          req.Message,
			Output:         "Error: " + err.Error(),
			TotalElapsedMs: elapsed.Milliseconds(),
			Status:         "error",
		})
		return
	}

	writeSSE(w, flusher, "stream", map[string]any{"content": resp.Content})
	writeSSE(w, flusher, "end", map[string]any{
		"metadata": Metadata{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			ElapsedMs:    elapsed.Milliseconds(),
		},
	})

	// Record trace
	s.traces.Add(TraceInfo{
		TraceID:           fmt.Sprintf("trace_%d", time.Now().UnixNano()),
		PipelineID:        "direct",
		PipelineName:      "Direct Chat",
		Timestamp:         start.UnixMilli(),
		Input:             req.Message,
		Output:            resp.Content,
		TotalElapsedMs:    elapsed.Milliseconds(),
		TotalInputTokens:  resp.Usage.PromptTokens,
		TotalOutputTokens: resp.Usage.CompletionTokens,
		TotalToolCalls:    0,
		Status:            "success",
	})
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}
	data["type"] = eventType
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()
}

func (s *Server) handlePipelineList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pipelines.List())
}

func (s *Server) handlePipelineSave(w http.ResponseWriter, r *http.Request) {
	var req SavePipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.pipelines.Save(PipelineInfo{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Nodes:       req.Nodes,
		Edges:       req.Edges,
		Layout:      req.Layout,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"success": true, "id": req.ID})
}

func (s *Server) handlePipelineDelete(w http.ResponseWriter, r *http.Request) {
	var req struct{ ID string `json:"id"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.pipelines.Delete(req.ID)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleTraceList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TraceListResponse{Traces: s.traces.List()})
}

func (s *Server) handleTraceGet(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	trace, ok := s.traces.Get(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TraceDetailResponse{Trace: trace, Spans: trace.Spans})
}

func (s *Server) handleTraceDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.traces.Delete(id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func (s *Server) handleMetricsSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.traces.Summary())
}

type runtimePipeline struct {
	Nodes []runtimeNode `json:"nodes"`
	Edges []runtimeEdge `json:"edges"`
}

type runtimeNode struct {
	ID     string   `json:"id"`
	Type   string   `json:"type"`
	Model  *string  `json:"model,omitempty"`
	Prompt *string  `json:"prompt,omitempty"`
	Tools  []string `json:"tools,omitempty"`
}

type runtimeEdge struct {
	From json.RawMessage `json:"from"`
	To   json.RawMessage `json:"to"`
}

func buildPipeline(rp runtimePipeline) *config.PipelineConfig {
	cfg := config.NewPipelineConfig("runtime", "Runtime Pipeline")

	for _, n := range rp.Nodes {
		nodeType, _ := config.ParseNodeType(n.Type)
		node := config.NewNodeConfig(n.ID, nodeType)
		if n.Prompt != nil {
			node.Prompt = *n.Prompt
		}
		if n.Model != nil {
			node.Model = core.DefaultModelConfig(*n.Model)
		}
		node.Tools = n.Tools
		cfg.AddNode(node)
	}

	for _, e := range rp.Edges {
		var from, to string
		json.Unmarshal(e.From, &from)
		json.Unmarshal(e.To, &to)

		if from == "" || to == "" {
			var fromObj, toObj map[string]string
			json.Unmarshal(e.From, &fromObj)
			json.Unmarshal(e.To, &toObj)
			from = fromObj["node"]
			to = toObj["node"]
		}

		cfg.AddEdge(from, to)
	}

	return cfg
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

var _ llm.Client = (*llm.UnifiedClient)(nil)
var _ tools.Tool = (tools.Tool)(nil)
