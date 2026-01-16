// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package scaffold generates Kairos project scaffolding.
package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

// Options configures project generation.
type Options struct {
	ProjectName string
	Module      string
	Archetype   string // assistant, tool-agent, coordinator, policy-heavy
	LLMProvider string // ollama, mock
	EnableMCP   bool
	EnableA2A   bool
	Corporate   bool // Include CI/CD, Dockerfile, observability stack
}

// Generate creates a new Kairos project at the given directory.
func Generate(dir string, opts Options) error {
	// Create directory structure
	dirs := []string{
		"cmd/agent",
		"internal/app",
		"internal/config",
		"internal/observability",
		"config",
	}

	// Add archetype-specific directories
	switch opts.Archetype {
	case "tool-agent":
		dirs = append(dirs, "internal/tools")
	case "coordinator":
		dirs = append(dirs, "internal/planner")
	case "policy-heavy":
		dirs = append(dirs, "internal/policies")
	}

	// Add corporate directories
	if opts.Corporate {
		dirs = append(dirs,
			".github/workflows",
			"deploy",
			"deploy/grafana/provisioning/dashboards",
			"deploy/grafana/provisioning/datasources",
		)
	}

	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(dir, d), 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Generate files from templates
	files := getFilesToGenerate(opts)

	for _, f := range files {
		if err := generateFile(dir, f, opts); err != nil {
			return fmt.Errorf("generating %s: %w", f.Path, err)
		}
		fmt.Printf("  Created: %s\n", f.Path)
	}

	return nil
}

type fileSpec struct {
	Path     string
	Template string
}

func getFilesToGenerate(opts Options) []fileSpec {
	files := []fileSpec{
		{"go.mod", goModTemplate},
		{".gitignore", gitignoreTemplate},
		{"Makefile", makefileTemplate},
		{"README.md", readmeTemplate},
		{"cmd/agent/main.go", mainTemplate},
		{"internal/config/config.go", configTemplate},
		{"internal/observability/otel.go", otelTemplate},
		{"internal/app/app.go", appTemplate},
		{"config/config.yaml", configYAMLTemplate},
		{"config/config.dev.yaml", configDevYAMLTemplate},
		{"config/config.prod.yaml", configProdYAMLTemplate},
	}

	// Add archetype-specific files
	switch opts.Archetype {
	case "tool-agent":
		files = append(files, fileSpec{"internal/tools/tools.go", toolsTemplate})
	case "coordinator":
		files = append(files, fileSpec{"internal/planner/planner.go", plannerTemplate})
	case "policy-heavy":
		files = append(files, fileSpec{"internal/policies/policies.go", policiesTemplate})
	}

	// Add corporate files (CI/CD, Docker, observability)
	if opts.Corporate {
		files = append(files,
			fileSpec{".github/workflows/ci.yaml", githubActionsTemplate},
			fileSpec{"Dockerfile", dockerfileTemplate},
			fileSpec{".dockerignore", dockerignoreTemplate},
			fileSpec{"docker-compose.yaml", dockerComposeTemplate},
			fileSpec{".golangci.yaml", golangciLintTemplate},
			fileSpec{"deploy/otel-collector-config.yaml", otelCollectorConfigTemplate},
			fileSpec{"deploy/prometheus.yaml", prometheusConfigTemplate},
		)
	}

	return files
}

func generateFile(dir string, spec fileSpec, opts Options) error {
	funcs := template.FuncMap{
		"title": func(s string) string {
			if len(s) == 0 {
				return s
			}
			return string(s[0]-32) + s[1:]
		},
	}

	tmpl, err := template.New(spec.Path).Funcs(funcs).Parse(spec.Template)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	path := filepath.Join(dir, spec.Path)
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, opts)
}
