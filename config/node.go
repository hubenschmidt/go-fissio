package config

import "github.com/hubenschmidt/go-fissio/core"

type EdgeEndpoint struct {
	Node string `json:"node"`
	Port string `json:"port,omitempty"`
}

type EdgeConfig struct {
	From      EdgeEndpoint `json:"from"`
	To        EdgeEndpoint `json:"to"`
	Type      EdgeType     `json:"type"`
	Condition string       `json:"condition,omitempty"`
}

type NodeConfig struct {
	ID          string           `json:"id"`
	Type        NodeType         `json:"type"`
	Prompt      string           `json:"prompt,omitempty"`
	Model       core.ModelConfig `json:"model,omitempty"`
	Tools       []string         `json:"tools,omitempty"`
	MaxIter     int              `json:"max_iter,omitempty"`
	NextNodes   []string         `json:"next_nodes,omitempty"`
	TargetNodes []string         `json:"target_nodes,omitempty"`
	Metadata    map[string]any   `json:"metadata,omitempty"`
}

func NewNodeConfig(id string, nodeType NodeType) *NodeConfig {
	cfg := &NodeConfig{
		ID:   id,
		Type: nodeType,
	}
	if nodeType == NodeWorker {
		cfg.MaxIter = 10
	}
	return cfg
}
