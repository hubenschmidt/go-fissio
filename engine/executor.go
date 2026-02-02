package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/hubenschmidt/go-fissio/config"
	"github.com/hubenschmidt/go-fissio/core"
	"github.com/hubenschmidt/go-fissio/llm"
	"github.com/hubenschmidt/go-fissio/tools"
)

type Executor struct {
	client   llm.Client
	resolver *ModelResolver
	registry *tools.Registry
}

func NewExecutor(client llm.Client, resolver *ModelResolver, registry *tools.Registry) *Executor {
	return &Executor{
		client:   client,
		resolver: resolver,
		registry: registry,
	}
}

func (e *Executor) Execute(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	start := time.Now()

	handlers := map[config.NodeType]func(context.Context, *config.NodeConfig, NodeInput) (NodeOutput, error){
		config.NodeLLM:          e.executeLLM,
		config.NodeWorker:       e.executeWorker,
		config.NodeRouter:       e.executeRouter,
		config.NodeGate:         e.executeGate,
		config.NodeAggregator:   e.executeAggregator,
		config.NodeOrchestrator: e.executeOrchestrator,
		config.NodeEvaluator:    e.executeEvaluator,
		config.NodeSynthesizer:  e.executeSynthesizer,
		config.NodeCoordinator:  e.executeCoordinator,
	}

	handler, ok := handlers[node.Type]
	if !ok {
		return NodeOutput{}, core.NewAgentError("executor.execute", node.ID, fmt.Errorf("unknown node type: %s", node.Type))
	}

	output, err := handler(ctx, node, input)
	if err != nil {
		return NodeOutput{}, err
	}

	output.NodeID = node.ID
	output.Duration = time.Since(start)
	return output, nil
}

func (e *Executor) executeLLM(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	model := e.resolver.ResolveModelName(node)
	resp, err := e.client.Chat(ctx, model, node.Prompt, input.Content)
	if err != nil {
		return NodeOutput{}, core.NewAgentError("executor.llm", node.ID, err)
	}

	return NodeOutput{
		Content:   resp.Content,
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}, nil
}

func (e *Executor) executeWorker(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	model := e.resolver.ResolveModelName(node)

	nodeTools, err := e.registry.GetMultiple(node.Tools)
	if err != nil {
		return NodeOutput{}, err
	}
	schemas := tools.ToSchemas(nodeTools)

	msgs := []core.Message{core.NewUserMessage(input.Content)}
	var totalIn, totalOut int

	maxIter := node.MaxIter
	if maxIter <= 0 {
		maxIter = 10
	}

	for i := 0; i < maxIter; i++ {
		resp, err := e.client.ChatWithTools(ctx, model, node.Prompt, msgs, schemas, nil)
		if err != nil {
			return NodeOutput{}, core.NewAgentError("executor.worker", node.ID, err)
		}

		totalIn += resp.Usage.PromptTokens
		totalOut += resp.Usage.CompletionTokens

		if !resp.HasToolCalls() {
			return NodeOutput{
				Content:   resp.Content,
				TokensIn:  totalIn,
				TokensOut: totalOut,
			}, nil
		}

		msgs = append(msgs, core.NewAssistantMessage(resp.Content))
		toolResults := e.executeToolCalls(ctx, resp.ToolCalls, nodeTools)

		for _, tr := range toolResults {
			msgs = append(msgs, core.NewToolMessage(tr.ToolCallID, tr.Content))
		}
	}

	return NodeOutput{}, core.NewAgentError("executor.worker", node.ID, core.ErrMaxIterations)
}

func (e *Executor) executeToolCalls(ctx context.Context, calls []core.ToolCall, nodeTools []tools.Tool) []core.ToolResult {
	toolMap := make(map[string]tools.Tool)
	for _, t := range nodeTools {
		toolMap[t.Name()] = t
	}

	results := make([]core.ToolResult, len(calls))
	for i, call := range calls {
		results[i] = e.executeSingleToolCall(ctx, call, toolMap)
	}

	return results
}

func (e *Executor) executeSingleToolCall(ctx context.Context, call core.ToolCall, toolMap map[string]tools.Tool) core.ToolResult {
	tool, ok := toolMap[call.Name]
	if !ok {
		return core.NewToolError(call.ID, fmt.Sprintf("tool not found: %s", call.Name))
	}

	result, err := tool.Execute(ctx, call.Arguments)
	if err != nil {
		return core.NewToolError(call.ID, err.Error())
	}
	return core.NewToolResult(call.ID, result)
}

func (e *Executor) executeRouter(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	model := e.resolver.ResolveModelName(node)
	prompt := node.Prompt + "\n\nAvailable routes: " + fmt.Sprintf("%v", node.NextNodes)

	resp, err := e.client.Chat(ctx, model, prompt, input.Content)
	if err != nil {
		return NodeOutput{}, core.NewAgentError("executor.router", node.ID, err)
	}

	return NodeOutput{
		Content:   resp.Content,
		NextNodes: []string{resp.Content},
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}, nil
}

func (e *Executor) executeGate(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	return NodeOutput{Content: input.Content}, nil
}

func (e *Executor) executeAggregator(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	return NodeOutput{Content: input.Content}, nil
}

func (e *Executor) executeOrchestrator(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	model := e.resolver.ResolveModelName(node)
	prompt := node.Prompt + "\n\nTarget nodes: " + fmt.Sprintf("%v", node.TargetNodes)

	resp, err := e.client.Chat(ctx, model, prompt, input.Content)
	if err != nil {
		return NodeOutput{}, core.NewAgentError("executor.orchestrator", node.ID, err)
	}

	return NodeOutput{
		Content:   resp.Content,
		NextNodes: node.TargetNodes,
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}, nil
}

func (e *Executor) executeEvaluator(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	model := e.resolver.ResolveModelName(node)
	resp, err := e.client.Chat(ctx, model, node.Prompt, input.Content)
	if err != nil {
		return NodeOutput{}, core.NewAgentError("executor.evaluator", node.ID, err)
	}

	return NodeOutput{
		Content:   resp.Content,
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}, nil
}

func (e *Executor) executeSynthesizer(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	model := e.resolver.ResolveModelName(node)
	resp, err := e.client.Chat(ctx, model, node.Prompt, input.Content)
	if err != nil {
		return NodeOutput{}, core.NewAgentError("executor.synthesizer", node.ID, err)
	}

	return NodeOutput{
		Content:   resp.Content,
		TokensIn:  resp.Usage.PromptTokens,
		TokensOut: resp.Usage.CompletionTokens,
	}, nil
}

func (e *Executor) executeCoordinator(ctx context.Context, node *config.NodeConfig, input NodeInput) (NodeOutput, error) {
	return NodeOutput{
		Content:   input.Content,
		NextNodes: node.TargetNodes,
	}, nil
}
