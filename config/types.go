package config

type NodeType int

const (
	NodeLLM NodeType = iota
	NodeWorker
	NodeRouter
	NodeGate
	NodeAggregator
	NodeOrchestrator
	NodeEvaluator
	NodeSynthesizer
	NodeCoordinator
)

var nodeTypeNames = map[NodeType]string{
	NodeLLM:          "llm",
	NodeWorker:       "worker",
	NodeRouter:       "router",
	NodeGate:         "gate",
	NodeAggregator:   "aggregator",
	NodeOrchestrator: "orchestrator",
	NodeEvaluator:    "evaluator",
	NodeSynthesizer:  "synthesizer",
	NodeCoordinator:  "coordinator",
}

var nodeTypeValues = map[string]NodeType{
	"llm":          NodeLLM,
	"worker":       NodeWorker,
	"router":       NodeRouter,
	"gate":         NodeGate,
	"aggregator":   NodeAggregator,
	"orchestrator": NodeOrchestrator,
	"evaluator":    NodeEvaluator,
	"synthesizer":  NodeSynthesizer,
	"coordinator":  NodeCoordinator,
}

func (n NodeType) String() string {
	if name, ok := nodeTypeNames[n]; ok {
		return name
	}
	return "unknown"
}

func ParseNodeType(s string) (NodeType, bool) {
	nt, ok := nodeTypeValues[s]
	return nt, ok
}

func (n NodeType) RequiresLLM() bool {
	llmTypes := map[NodeType]bool{
		NodeLLM:          true,
		NodeWorker:       true,
		NodeRouter:       true,
		NodeOrchestrator: true,
		NodeEvaluator:    true,
		NodeSynthesizer:  true,
	}
	return llmTypes[n]
}

type EdgeType int

const (
	EdgeDefault EdgeType = iota
	EdgeConditional
	EdgeLoop
)

var edgeTypeNames = map[EdgeType]string{
	EdgeDefault:     "default",
	EdgeConditional: "conditional",
	EdgeLoop:        "loop",
}

func (e EdgeType) String() string {
	if name, ok := edgeTypeNames[e]; ok {
		return name
	}
	return "unknown"
}
