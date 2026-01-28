// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jllopis/kairos/pkg/planner"
)

type graphResult struct {
	Format  string `json:"format"`
	Content string `json:"content"`
	GraphID string `json:"graph_id,omitempty"`
	Nodes   int    `json:"nodes"`
	Edges   int    `json:"edges"`
}

func runGraph(global globalFlags, args []string) {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	output := fs.String("output", "mermaid", "Output format: mermaid, dot, json")
	graphPath := fs.String("path", "", "Path to graph YAML/JSON file")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}

	// Determine graph path
	path := *graphPath
	if path == "" {
		fatal(fmt.Errorf("no graph path specified; use --path <file>"))
	}

	// Load graph
	graph, err := loadGraph(path)
	if err != nil {
		fatal(err)
	}

	if err := graph.Validate(); err != nil {
		fatal(fmt.Errorf("invalid graph: %w", err))
	}

	result := graphResult{
		Format:  *output,
		GraphID: graph.ID,
		Nodes:   len(graph.Nodes),
		Edges:   len(graph.Edges),
	}

	switch *output {
	case "mermaid":
		result.Content = toMermaid(graph)
	case "dot":
		result.Content = toDot(graph)
	case "json":
		jsonBytes, err := planner.MarshalJSON(graph, true)
		if err != nil {
			fatal(err)
		}
		result.Content = string(jsonBytes)
	default:
		fatal(fmt.Errorf("unknown output format %q; use mermaid, dot, or json", *output))
	}

	if global.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fatal(err)
		}
		return
	}

	fmt.Println(result.Content)
}

func loadGraph(path string) (*planner.Graph, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return planner.ParseYAML(data)
	case ".json":
		return planner.ParseJSON(data)
	default:
		// Try YAML first, then JSON
		graph, err := planner.ParseYAML(data)
		if err == nil {
			return graph, nil
		}
		return planner.ParseJSON(data)
	}
}

func toMermaid(g *planner.Graph) string {
	var sb strings.Builder
	sb.WriteString("graph TD\n")

	// Nodes
	for id, node := range g.Nodes {
		label := id
		if node.Type != "" {
			label = fmt.Sprintf("%s[%s: %s]", id, id, node.Type)
		} else {
			label = fmt.Sprintf("%s[%s]", id, id)
		}
		sb.WriteString(fmt.Sprintf("    %s\n", label))
	}

	// Edges
	for _, edge := range g.Edges {
		if edge.Condition != "" && edge.Condition != "default" && edge.Condition != "always" {
			sb.WriteString(fmt.Sprintf("    %s -->|%s| %s\n", edge.From, edge.Condition, edge.To))
		} else {
			sb.WriteString(fmt.Sprintf("    %s --> %s\n", edge.From, edge.To))
		}
	}

	// Mark start node
	if g.Start != "" {
		sb.WriteString(fmt.Sprintf("    style %s fill:#90EE90\n", g.Start))
	}

	return sb.String()
}

func toDot(g *planner.Graph) string {
	var sb strings.Builder
	sb.WriteString("digraph G {\n")
	sb.WriteString("    rankdir=TB;\n")
	sb.WriteString("    node [shape=box, style=rounded];\n")

	// Nodes
	for id, node := range g.Nodes {
		label := id
		if node.Type != "" {
			label = fmt.Sprintf("%s\\n(%s)", id, node.Type)
		}
		attrs := fmt.Sprintf("label=\"%s\"", label)

		// Highlight start node
		if id == g.Start {
			attrs += ", style=\"rounded,filled\", fillcolor=\"#90EE90\""
		}

		sb.WriteString(fmt.Sprintf("    %q [%s];\n", id, attrs))
	}

	// Edges
	for _, edge := range g.Edges {
		attrs := ""
		if edge.Condition != "" && edge.Condition != "default" && edge.Condition != "always" {
			attrs = fmt.Sprintf(" [label=\"%s\"]", edge.Condition)
		}
		sb.WriteString(fmt.Sprintf("    %q -> %q%s;\n", edge.From, edge.To, attrs))
	}

	sb.WriteString("}\n")
	return sb.String()
}
