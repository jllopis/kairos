// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// Connector defines the minimal interface for tool-generating connectors.
type Connector interface {
	Tools() []core.Tool
	Execute(ctx context.Context, toolName string, args map[string]any) (any, error)
}

type toolAdapter struct {
	name       string
	definition llm.Tool
	executor   interface {
		Execute(ctx context.Context, toolName string, args map[string]any) (any, error)
	}
}

func (t *toolAdapter) Name() string {
	return t.name
}

func (t *toolAdapter) ToolDefinition() llm.Tool {
	return t.definition
}

func (t *toolAdapter) Call(ctx context.Context, input any) (any, error) {
	if t.executor == nil {
		return nil, errors.New("connector tool executor is nil")
	}
	args, err := normalizeToolArgs(input)
	if err != nil {
		return nil, err
	}
	return t.executor.Execute(ctx, t.name, args)
}

func coreToolsFromDefinitions(defs []llm.Tool, exec interface {
	Execute(ctx context.Context, toolName string, args map[string]any) (any, error)
}) []core.Tool {
	if exec == nil || len(defs) == 0 {
		return nil
	}
	tools := make([]core.Tool, 0, len(defs))
	for _, def := range defs {
		name := strings.TrimSpace(def.Function.Name)
		if name == "" {
			continue
		}
		if def.Type == "" {
			def.Type = llm.ToolTypeFunction
		}
		def.Function.Name = name
		tools = append(tools, &toolAdapter{
			name:       name,
			definition: def,
			executor:   exec,
		})
	}
	return tools
}

func normalizeToolArgs(input any) (map[string]any, error) {
	switch value := input.(type) {
	case nil:
		return map[string]any{}, nil
	case map[string]any:
		return value, nil
	case json.RawMessage:
		var decoded map[string]any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, fmt.Errorf("connector tool args: invalid JSON: %w", err)
		}
		return decoded, nil
	case []byte:
		var decoded map[string]any
		if err := json.Unmarshal(value, &decoded); err != nil {
			return nil, fmt.Errorf("connector tool args: invalid JSON: %w", err)
		}
		return decoded, nil
	case string:
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return map[string]any{}, nil
		}
		if strings.HasPrefix(trimmed, "{") {
			var decoded map[string]any
			if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
				return decoded, nil
			}
		}
		return map[string]any{"input": value}, nil
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("connector tool args: unsupported type %T", input)
		}
		var decoded map[string]any
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			return nil, fmt.Errorf("connector tool args: invalid JSON after marshal: %w", err)
		}
		return decoded, nil
	}
}
