package config

import "github.com/hubenschmidt/go-fissio/core"

type PipelineBuilder struct {
	config *PipelineConfig
}

type NodeBuilder struct {
	pipeline *PipelineBuilder
	node     *NodeConfig
}

func NewPipeline(id, name string) *PipelineBuilder {
	return &PipelineBuilder{
		config: NewPipelineConfig(id, name),
	}
}

func (b *PipelineBuilder) Description(desc string) *PipelineBuilder {
	b.config.Description = desc
	return b
}

func (b *PipelineBuilder) Node(id string, nodeType NodeType) *NodeBuilder {
	node := NewNodeConfig(id, nodeType)
	return &NodeBuilder{pipeline: b, node: node}
}

func (b *PipelineBuilder) Edge(from, to string) *PipelineBuilder {
	b.config.AddEdge(from, to)
	return b
}

func (b *PipelineBuilder) ConditionalEdge(from, to, condition string) *PipelineBuilder {
	b.config.Edges = append(b.config.Edges, EdgeConfig{
		From:      EdgeEndpoint{Node: from},
		To:        EdgeEndpoint{Node: to},
		Type:      EdgeConditional,
		Condition: condition,
	})
	return b
}

func (b *PipelineBuilder) EntryNode(id string) *PipelineBuilder {
	b.config.EntryNode = id
	return b
}

func (b *PipelineBuilder) Build() *PipelineConfig {
	return b.config
}

func (n *NodeBuilder) Prompt(prompt string) *NodeBuilder {
	n.node.Prompt = prompt
	return n
}

func (n *NodeBuilder) Model(name string) *NodeBuilder {
	n.node.Model = core.DefaultModelConfig(name)
	return n
}

func (n *NodeBuilder) ModelConfig(cfg core.ModelConfig) *NodeBuilder {
	n.node.Model = cfg
	return n
}

func (n *NodeBuilder) Tools(names ...string) *NodeBuilder {
	n.node.Tools = append(n.node.Tools, names...)
	return n
}

func (n *NodeBuilder) MaxIterations(max int) *NodeBuilder {
	n.node.MaxIter = max
	return n
}

func (n *NodeBuilder) NextNodes(nodes ...string) *NodeBuilder {
	n.node.NextNodes = append(n.node.NextNodes, nodes...)
	return n
}

func (n *NodeBuilder) TargetNodes(nodes ...string) *NodeBuilder {
	n.node.TargetNodes = append(n.node.TargetNodes, nodes...)
	return n
}

func (n *NodeBuilder) Meta(key string, val any) *NodeBuilder {
	if n.node.Metadata == nil {
		n.node.Metadata = make(map[string]any)
	}
	n.node.Metadata[key] = val
	return n
}

func (n *NodeBuilder) Done() *PipelineBuilder {
	n.pipeline.config.AddNode(n.node)
	return n.pipeline
}
