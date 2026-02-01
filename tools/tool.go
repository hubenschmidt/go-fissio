package tools

import (
	"context"
	"encoding/json"

	"github.com/hubenschmidt/go-fissio/core"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

func ToSchema(t Tool) core.ToolSchema {
	return core.ToolSchema{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}

func ToSchemas(tools []Tool) []core.ToolSchema {
	schemas := make([]core.ToolSchema, len(tools))
	for i, t := range tools {
		schemas[i] = ToSchema(t)
	}
	return schemas
}
