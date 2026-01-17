// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jllopis/kairos/pkg/llm"
)

// SkillTool wraps a SkillSpec as an executable tool for LLM tool calling.
// It implements progressive disclosure: the LLM sees metadata initially,
// and receives the full instructions (Body) when it invokes the skill.
type SkillTool struct {
	spec      SkillSpec
	activated bool
}

// NewSkillTool creates a SkillTool from a SkillSpec.
func NewSkillTool(spec SkillSpec) *SkillTool {
	return &SkillTool{spec: spec}
}

// Name returns the skill name.
func (s *SkillTool) Name() string {
	return s.spec.Name
}

// Call executes the skill, returning its instructions (Body) and optionally
// loading referenced resources. This implements the "activation" phase of
// progressive disclosure.
func (s *SkillTool) Call(ctx context.Context, input any) (any, error) {
	s.activated = true

	// Parse input to check for resource requests
	var req SkillRequest
	if input != nil {
		switch v := input.(type) {
		case string:
			if err := json.Unmarshal([]byte(v), &req); err != nil {
				// If not JSON, treat as simple activation
				req.Action = "activate"
			}
		case map[string]any:
			if action, ok := v["action"].(string); ok {
				req.Action = action
			}
			if resource, ok := v["resource"].(string); ok {
				req.Resource = resource
			}
		}
	}

	if req.Action == "" {
		req.Action = "activate"
	}

	switch req.Action {
	case "activate":
		return s.activate()
	case "load_resource":
		return s.loadResource(req.Resource)
	case "list_resources":
		return s.listResources()
	default:
		return s.activate()
	}
}

// SkillRequest defines the input structure for skill invocation.
type SkillRequest struct {
	Action   string `json:"action,omitempty"`   // activate, load_resource, list_resources
	Resource string `json:"resource,omitempty"` // path to resource file
}

// SkillResponse contains the skill activation result.
type SkillResponse struct {
	Name         string   `json:"name"`
	Instructions string   `json:"instructions"`
	Resources    []string `json:"resources,omitempty"`
}

// activate returns the skill body (instructions) for the LLM.
func (s *SkillTool) activate() (*SkillResponse, error) {
	resources, _ := s.listResources()
	resourceList, _ := resources.([]string)

	return &SkillResponse{
		Name:         s.spec.Name,
		Instructions: s.spec.Body,
		Resources:    resourceList,
	}, nil
}

// loadResource loads a specific resource file from the skill directory.
func (s *SkillTool) loadResource(resourcePath string) (string, error) {
	if resourcePath == "" {
		return "", fmt.Errorf("resource path is required")
	}

	// Security: prevent directory traversal
	cleanPath := filepath.Clean(resourcePath)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("invalid resource path: %s", resourcePath)
	}

	fullPath := filepath.Join(s.spec.Dir, cleanPath)

	// Verify the path is still within the skill directory
	absDir, _ := filepath.Abs(s.spec.Dir)
	absPath, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absPath, absDir) {
		return "", fmt.Errorf("resource path outside skill directory")
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to load resource %s: %w", resourcePath, err)
	}

	return string(data), nil
}

// listResources returns available resources in the skill directory.
func (s *SkillTool) listResources() (any, error) {
	var resources []string

	subdirs := []string{"scripts", "references", "assets"}
	for _, subdir := range subdirs {
		dirPath := filepath.Join(s.spec.Dir, subdir)
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			continue // Directory doesn't exist, skip
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				resources = append(resources, filepath.Join(subdir, entry.Name()))
			}
		}
	}

	return resources, nil
}

// ToolDefinition returns the LLM tool definition for this skill.
// This is what the LLM sees initially (metadata only, not the body).
func (s *SkillTool) ToolDefinition() llm.Tool {
	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        s.spec.Name,
			Description: s.spec.Description,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"enum":        []string{"activate", "load_resource", "list_resources"},
						"description": "Action to perform: 'activate' to get skill instructions, 'load_resource' to load a specific file, 'list_resources' to see available resources",
						"default":     "activate",
					},
					"resource": map[string]any{
						"type":        "string",
						"description": "Path to resource file (for load_resource action)",
					},
				},
				"required": []string{},
			},
		},
	}
}

// IsActivated returns whether this skill has been invoked.
func (s *SkillTool) IsActivated() bool {
	return s.activated
}

// Spec returns the underlying SkillSpec.
func (s *SkillTool) Spec() SkillSpec {
	return s.spec
}

// LoadToolsFromDir loads skills from a directory and returns them as SkillTools.
func LoadToolsFromDir(root string) ([]*SkillTool, error) {
	specs, err := LoadDir(root)
	if err != nil {
		return nil, err
	}

	tools := make([]*SkillTool, len(specs))
	for i, spec := range specs {
		tools[i] = NewSkillTool(spec)
	}

	return tools, nil
}
