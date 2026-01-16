// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jllopis/kairos/cmd/kairos/scaffold"
)

func runInit(global *globalFlags, args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)

	module := fs.String("module", "", "Go module path (required, e.g., github.com/myorg/my-agent)")
	archetype := fs.String("type", "assistant", "Project archetype: assistant, tool-agent, coordinator, policy-heavy")
	llmProvider := fs.String("llm", "ollama", "Default LLM provider: ollama, mock")
	enableMCP := fs.Bool("mcp", false, "Include MCP server configuration")
	enableA2A := fs.Bool("a2a", false, "Include A2A endpoint configuration")
	corporate := fs.Bool("corporate", false, "Include CI/CD, Dockerfile, and observability stack")
	overwrite := fs.Bool("overwrite", false, "Overwrite existing files")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: kairos init <directory> [flags]

Generate a new Kairos agent project with recommended structure.

Arguments:
  directory    Target directory for the new project

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Archetypes:
  assistant     Basic agent with memory and conversation (default)
  tool-agent    Agent with local tools and MCP integration
  coordinator   Multi-agent coordinator with planner
  policy-heavy  Agent with strict governance policies

Corporate Template (--corporate):
  Adds enterprise-ready infrastructure:
  - GitHub Actions CI/CD pipeline
  - Optimized multi-stage Dockerfile
  - docker-compose.yaml with observability stack
  - OpenTelemetry Collector configuration
  - Prometheus and Grafana setup
  - golangci-lint configuration

Examples:
  kairos init my-agent --module github.com/myorg/my-agent
  kairos init my-agent --module github.com/myorg/my-agent --type tool-agent --mcp
  kairos init my-agent --module github.com/myorg/my-agent --corporate
`)
	}

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: directory argument required")
		fs.Usage()
		os.Exit(1)
	}

	dir := fs.Arg(0)

	if *module == "" {
		fmt.Fprintln(os.Stderr, "Error: --module flag is required")
		fs.Usage()
		os.Exit(1)
	}

	// Validate archetype
	validTypes := map[string]bool{
		"assistant":    true,
		"tool-agent":   true,
		"coordinator":  true,
		"policy-heavy": true,
	}
	if !validTypes[*archetype] {
		fmt.Fprintf(os.Stderr, "Error: invalid --type %q. Valid options: assistant, tool-agent, coordinator, policy-heavy\n", *archetype)
		os.Exit(1)
	}

	// Get absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid directory path: %v\n", err)
		os.Exit(1)
	}

	// Check if directory exists
	if _, err := os.Stat(absDir); err == nil && !*overwrite {
		fmt.Fprintf(os.Stderr, "Error: directory %q already exists. Use --overwrite to replace.\n", dir)
		os.Exit(1)
	}

	// Extract project name from directory
	projectName := filepath.Base(absDir)

	opts := scaffold.Options{
		ProjectName: projectName,
		Module:      *module,
		Archetype:   *archetype,
		LLMProvider: *llmProvider,
		EnableMCP:   *enableMCP,
		EnableA2A:   *enableA2A,
		Corporate:   *corporate,
	}

	fmt.Printf("Creating Kairos project %q...\n", projectName)
	fmt.Printf("  Module:    %s\n", *module)
	fmt.Printf("  Archetype: %s\n", *archetype)
	fmt.Printf("  LLM:       %s\n", *llmProvider)
	if *corporate {
		fmt.Println("  Corporate: enabled (CI/CD, Docker, observability)")
	}

	if err := scaffold.Generate(absDir, opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating project: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("✅ Project created successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Printf("  cd %s\n", dir)
	fmt.Println("  go mod tidy")
	if *corporate {
		fmt.Println("  docker-compose up -d  # Start observability stack")
	}
	fmt.Println("  make run")
}
