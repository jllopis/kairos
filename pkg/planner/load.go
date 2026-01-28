// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package planner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadGraph loads a planner graph from a YAML or JSON file.
func LoadGraph(path string) (*Graph, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("graph path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return ParseJSON(data)
	case ".yaml", ".yml":
		return ParseYAML(data)
	default:
		return parseGraphAuto(data)
	}
}

func parseGraphAuto(data []byte) (*Graph, error) {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		if graph, err := ParseJSON(data); err == nil {
			return graph, nil
		}
	}
	if graph, err := ParseYAML(data); err == nil {
		return graph, nil
	}
	if graph, err := ParseJSON(data); err == nil {
		return graph, nil
	}
	return nil, fmt.Errorf("unsupported graph format")
}
