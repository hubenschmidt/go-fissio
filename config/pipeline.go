package config

import (
	"encoding/json"
	"os"
)

type PipelineConfig struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Nodes       []*NodeConfig `json:"nodes"`
	Edges       []EdgeConfig  `json:"edges"`
	EntryNode   string        `json:"entry_node,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func NewPipelineConfig(id, name string) *PipelineConfig {
	return &PipelineConfig{
		ID:    id,
		Name:  name,
		Nodes: make([]*NodeConfig, 0),
		Edges: make([]EdgeConfig, 0),
	}
}

func (p *PipelineConfig) AddNode(node *NodeConfig) *PipelineConfig {
	p.Nodes = append(p.Nodes, node)
	return p
}

func (p *PipelineConfig) AddEdge(from, to string) *PipelineConfig {
	p.Edges = append(p.Edges, EdgeConfig{
		From: EdgeEndpoint{Node: from},
		To:   EdgeEndpoint{Node: to},
		Type: EdgeDefault,
	})
	return p
}

func (p *PipelineConfig) GetNode(id string) *NodeConfig {
	for _, n := range p.Nodes {
		if n.ID == id {
			return n
		}
	}
	return nil
}

func (p *PipelineConfig) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

func LoadPipeline(path string) (*PipelineConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg PipelineConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (p *PipelineConfig) Save(path string) error {
	data, err := p.ToJSON()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
