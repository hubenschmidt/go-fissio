package tools

import (
	"sync"

	"github.com/hubenschmidt/go-fissio/core"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) GetMultiple(names []string) ([]Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Tool, 0, len(names))
	for _, name := range names {
		t, ok := r.tools[name]
		if !ok {
			return nil, core.NewAgentError("registry.get", "", core.ErrToolNotFound)
		}
		result = append(result, t)
	}
	return result, nil
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

func (r *Registry) Schemas(names []string) ([]core.ToolSchema, error) {
	tools, err := r.GetMultiple(names)
	if err != nil {
		return nil, err
	}
	return ToSchemas(tools), nil
}

var DefaultRegistry = NewRegistry()

func Register(t Tool) {
	DefaultRegistry.Register(t)
}

func Get(name string) (Tool, bool) {
	return DefaultRegistry.Get(name)
}
