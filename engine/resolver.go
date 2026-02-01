package engine

import (
	"github.com/hubenschmidt/go-fissio/config"
	"github.com/hubenschmidt/go-fissio/core"
)

type ModelResolver struct {
	defaultModel core.ModelConfig
	overrides    map[string]core.ModelConfig
}

func NewModelResolver(defaultModel core.ModelConfig) *ModelResolver {
	return &ModelResolver{
		defaultModel: defaultModel,
		overrides:    make(map[string]core.ModelConfig),
	}
}

func (r *ModelResolver) SetOverride(nodeID string, model core.ModelConfig) {
	r.overrides[nodeID] = model
}

func (r *ModelResolver) Resolve(node *config.NodeConfig) core.ModelConfig {
	if override, ok := r.overrides[node.ID]; ok {
		return override
	}

	if node.Model.Name != "" {
		return node.Model
	}

	return r.defaultModel
}

func (r *ModelResolver) ResolveModelName(node *config.NodeConfig) string {
	return r.Resolve(node).Name
}
