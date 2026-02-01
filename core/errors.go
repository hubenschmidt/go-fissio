package core

import (
	"errors"
	"fmt"
)

var (
	ErrNodeNotFound     = errors.New("node not found")
	ErrToolNotFound     = errors.New("tool not found")
	ErrModelNotFound    = errors.New("model not found")
	ErrInvalidConfig    = errors.New("invalid configuration")
	ErrExecutionFailed  = errors.New("execution failed")
	ErrMaxIterations    = errors.New("max iterations exceeded")
	ErrCyclicDependency = errors.New("cyclic dependency detected")
	ErrInvalidEdge      = errors.New("invalid edge configuration")
	ErrTimeout          = errors.New("operation timed out")
	ErrLLMRequest       = errors.New("LLM request failed")
)

type AgentError struct {
	Op      string
	Node    string
	Err     error
	Context map[string]any
}

func (e *AgentError) Error() string {
	if e.Node != "" {
		return fmt.Sprintf("%s [node=%s]: %v", e.Op, e.Node, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *AgentError) Unwrap() error {
	return e.Err
}

func NewAgentError(op, node string, err error) *AgentError {
	return &AgentError{Op: op, Node: node, Err: err}
}

func WithContext(err *AgentError, key string, val any) *AgentError {
	if err.Context == nil {
		err.Context = make(map[string]any)
	}
	err.Context[key] = val
	return err
}
