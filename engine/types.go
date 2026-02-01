package engine

import "time"

type NodeInput struct {
	NodeID   string         `json:"node_id"`
	Content  string         `json:"content"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Sources  []string       `json:"sources,omitempty"`
}

type NodeOutput struct {
	NodeID    string         `json:"node_id"`
	Content   string         `json:"content"`
	NextNodes []string       `json:"next_nodes,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	TokensIn  int            `json:"tokens_in,omitempty"`
	TokensOut int            `json:"tokens_out,omitempty"`
	Duration  time.Duration  `json:"duration,omitempty"`
}

type EngineOutput struct {
	Success   bool                  `json:"success"`
	FinalNode string                `json:"final_node"`
	Content   string                `json:"content"`
	Outputs   map[string]NodeOutput `json:"outputs"`
	Error     error                 `json:"error,omitempty"`
	Duration  time.Duration         `json:"duration"`
}

type ExecutionContext struct {
	Input     NodeInput
	History   []NodeOutput
	Variables map[string]any
}

func NewExecutionContext(input NodeInput) *ExecutionContext {
	return &ExecutionContext{
		Input:     input,
		History:   make([]NodeOutput, 0),
		Variables: make(map[string]any),
	}
}

func (c *ExecutionContext) AddOutput(out NodeOutput) {
	c.History = append(c.History, out)
}

func (c *ExecutionContext) GetOutput(nodeID string) (NodeOutput, bool) {
	for i := len(c.History) - 1; i >= 0; i-- {
		if c.History[i].NodeID == nodeID {
			return c.History[i], true
		}
	}
	return NodeOutput{}, false
}
