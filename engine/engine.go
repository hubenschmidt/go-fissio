package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hubenschmidt/go-fissio/config"
	"github.com/hubenschmidt/go-fissio/core"
	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/monitor"
	"github.com/hubenschmidt/go-fissio/tools"
)

type Engine struct {
	pipeline  *config.PipelineConfig
	executor  *Executor
	collector monitor.MetricsCollector
	nodeMap   map[string]*config.NodeConfig
	edges     map[string][]string
}

type EngineConfig struct {
	Client       llm.Client
	Registry     *tools.Registry
	Resolver     *ModelResolver
	Collector    monitor.MetricsCollector
}

func NewEngine(pipeline *config.PipelineConfig, cfg EngineConfig) *Engine {
	registry := cfg.Registry
	if registry == nil {
		registry = tools.DefaultRegistry
	}

	resolver := cfg.Resolver
	if resolver == nil {
		resolver = NewModelResolver(core.DefaultModelConfig("gpt-4"))
	}

	nodeMap := make(map[string]*config.NodeConfig)
	for _, n := range pipeline.Nodes {
		nodeMap[n.ID] = n
	}

	edges := make(map[string][]string)
	for _, e := range pipeline.Edges {
		edges[e.From.Node] = append(edges[e.From.Node], e.To.Node)
	}

	return &Engine{
		pipeline:  pipeline,
		executor:  NewExecutor(cfg.Client, resolver, registry),
		collector: cfg.Collector,
		nodeMap:   nodeMap,
		edges:     edges,
	}
}

func (e *Engine) Run(ctx context.Context, input string) (*EngineOutput, error) {
	start := time.Now()

	log.Println("╔══════════════════════════════════════════════════════════════")
	log.Printf("║ PIPELINE: %s", e.pipeline.Name)
	log.Printf("║ Input: %.50s...", input)
	log.Println("╠══════════════════════════════════════════════════════════════")

	entryNode := e.pipeline.EntryNode
	if entryNode == "" {
		entryNode = e.findEntryNode()
	}

	if entryNode == "" {
		return nil, core.NewAgentError("engine.run", "", core.ErrNodeNotFound)
	}

	execCtx := NewExecutionContext(NodeInput{Content: input})
	outputs := make(map[string]NodeOutput)
	var spans []Span
	step := 0

	currentNodes := []string{entryNode}
	visited := make(map[string]bool)

	for len(currentNodes) > 0 {
		var nextNodes []string

		for _, nodeID := range currentNodes {
			if visited[nodeID] {
				continue
			}
			visited[nodeID] = true

			node, ok := e.nodeMap[nodeID]
			if !ok {
				continue
			}

			step++
			model := node.Model.Name
			if model == "" {
				model = "default"
			}
			log.Println("╠──────────────────────────────────────────────────────────────")
			log.Printf("║ [%d] NODE: %s (%s)", step, nodeID, node.Type)
			log.Printf("║     Model: %s", model)
			if len(node.Tools) > 0 {
				log.Printf("║     Tools: %v", node.Tools)
			}

			nodeInput := e.buildNodeInput(nodeID, execCtx)
			nodeStart := time.Now()
			output, err := e.executor.Execute(ctx, node, nodeInput)
			nodeEnd := time.Now()

			if err != nil {
				log.Printf("║     ✗ Error: %v", err)
				log.Println("╚══════════════════════════════════════════════════════════════")
				return &EngineOutput{
					Success:  false,
					Error:    err,
					Outputs:  outputs,
					Spans:    spans,
					Duration: time.Since(start),
				}, err
			}

			log.Printf("║     ✓ Completed in %v", nodeEnd.Sub(nodeStart))
			log.Printf("║     ← Response: %d chars, %d/%d tokens", len(output.Content), output.TokensIn, output.TokensOut)

			step++
			spans = append(spans, Span{
				SpanID:       fmt.Sprintf("span_%d", step),
				NodeID:       nodeID,
				NodeType:     string(node.Type),
				StartTime:    nodeStart.UnixMilli(),
				EndTime:      nodeEnd.UnixMilli(),
				Input:        nodeInput.Content,
				Output:       output.Content,
				InputTokens:  output.TokensIn,
				OutputTokens: output.TokensOut,
				Duration:     nodeEnd.Sub(nodeStart),
			})

			outputs[nodeID] = output
			execCtx.AddOutput(output)
			e.recordMetrics(nodeID, output)

			nextNodes = append(nextNodes, e.getNextNodes(nodeID, output)...)
		}

		currentNodes = nextNodes
	}

	finalOutput := e.findFinalOutput(execCtx)
	log.Println("╠══════════════════════════════════════════════════════════════")
	log.Printf("║ Pipeline complete in %v", time.Since(start))
	log.Printf("║ Output: %d chars", len(finalOutput.Content))
	log.Println("╚══════════════════════════════════════════════════════════════")

	return &EngineOutput{
		Success:   true,
		FinalNode: finalOutput.NodeID,
		Content:   finalOutput.Content,
		Outputs:   outputs,
		Spans:     spans,
		Duration:  time.Since(start),
	}, nil
}

func (e *Engine) findEntryNode() string {
	hasIncoming := make(map[string]bool)
	for _, edge := range e.pipeline.Edges {
		hasIncoming[edge.To.Node] = true
	}

	for _, node := range e.pipeline.Nodes {
		if !hasIncoming[node.ID] {
			return node.ID
		}
	}

	if len(e.pipeline.Nodes) > 0 {
		return e.pipeline.Nodes[0].ID
	}
	return ""
}

func (e *Engine) buildNodeInput(nodeID string, ctx *ExecutionContext) NodeInput {
	var sources []string
	var content string

	for from, tos := range e.edges {
		for _, to := range tos {
			if to != nodeID {
				continue
			}
			sources = append(sources, from)
			if out, ok := ctx.GetOutput(from); ok {
				if content != "" {
					content += "\n\n"
				}
				content += out.Content
			}
		}
	}

	if content == "" {
		content = ctx.Input.Content
	}

	return NodeInput{
		NodeID:  nodeID,
		Content: content,
		Sources: sources,
	}
}

func (e *Engine) getNextNodes(nodeID string, output NodeOutput) []string {
	if len(output.NextNodes) > 0 {
		return output.NextNodes
	}
	return e.edges[nodeID]
}

func (e *Engine) findFinalOutput(ctx *ExecutionContext) NodeOutput {
	if len(ctx.History) == 0 {
		return NodeOutput{}
	}
	return ctx.History[len(ctx.History)-1]
}

func (e *Engine) recordMetrics(nodeID string, output NodeOutput) {
	if e.collector == nil {
		return
	}
	e.collector.Record(monitor.NodeMetrics{
		NodeID:    nodeID,
		TokensIn:  output.TokensIn,
		TokensOut: output.TokensOut,
		Duration:  output.Duration,
		Success:   true,
	})
}
