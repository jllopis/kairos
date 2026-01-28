// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package scaffold

// Base templates used by all archetypes

const goModTemplate = `module {{.Module}}

go 1.22

require (
	github.com/jllopis/kairos v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)
`

const gitignoreTemplate = `/bin/
*.out
*.exe

.idea/
.vscode/
.DS_Store

.env
config/*.local.yaml
`

const makefileTemplate = `.PHONY: run run-dev run-prod build test tidy

run:
	go run ./cmd/agent --config ./config/config.yaml

run-dev:
	go run ./cmd/agent --config ./config/config.yaml --env dev

run-prod:
	go run ./cmd/agent --config ./config/config.yaml --env prod

build:
	go build -o bin/agent ./cmd/agent

test:
	go test ./...

tidy:
	go mod tidy
`

const readmeTemplate = `# {{.ProjectName}}

A Kairos agent ({{.Archetype}} archetype).

## Quick Start

### Prerequisites

- Go 1.22+
{{- if eq .LLMProvider "ollama"}}
- Ollama running locally:
  ` + "`" + `bash
  ollama serve
  ollama pull llama3.1
  ` + "`" + `
{{- end}}

### Run

` + "```" + `bash
# Install dependencies
go mod tidy

# Development mode (mock LLM)
make run-dev

# With real LLM
make run
` + "```" + `

## Project Structure

` + "```" + `
{{.ProjectName}}/
├── cmd/agent/main.go           # Entrypoint
├── internal/
│   ├── app/app.go              # Component wiring
│   ├── config/config.go        # Configuration loader
│   └── observability/otel.go   # OTEL setup
├── config/
│   ├── config.yaml             # Base configuration
│   ├── config.dev.yaml         # Development overrides
│   └── config.prod.yaml        # Production overrides
└── Makefile
` + "```" + `

## Configuration

Edit ` + "`" + `config/config.yaml` + "`" + ` to customize:

- LLM provider and model
- Memory backend
- Governance policies
- Telemetry export

## Next Steps

{{- if eq .Archetype "assistant"}}
- Add tools with ` + "`" + `agent.WithTools()` + "`" + `
- Enable memory persistence
- Configure governance policies
{{- else if eq .Archetype "tool-agent"}}
- Add your tools in ` + "`" + `internal/tools/tools.go` + "`" + `
- Configure MCP servers in config.yaml
{{- else if eq .Archetype "coordinator"}}
- Define your planner graph in ` + "`" + `internal/planner/planner.go` + "`" + `
- Configure A2A endpoints for agent discovery
{{- else if eq .Archetype "policy-heavy"}}
- Review policies in ` + "`" + `internal/policies/policies.go` + "`" + `
- Add compliance rules for your use case
{{- end}}

## Documentation

- [Kairos Documentation](https://github.com/jllopis/kairos)
- [Examples](https://github.com/jllopis/kairos/tree/main/examples)
`

const mainTemplate = `// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"{{.Module}}/internal/app"
	"{{.Module}}/internal/config"
)

func main() {
	cfgPath := flag.String("config", "./config/config.yaml", "path to config file")
	envOverride := flag.String("env", "", "environment override (dev, prod)")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(*cfgPath, *envOverride)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	if err := application.Run(ctx); err != nil {
		log.Fatalf("app error: %v", err)
	}
}
`

const configTemplate = `// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Name     string ` + "`" + `yaml:"name"` + "`" + `
		LogLevel string ` + "`" + `yaml:"log_level"` + "`" + `
	} ` + "`" + `yaml:"app"` + "`" + `

	LLM struct {
		Provider string ` + "`" + `yaml:"provider"` + "`" + `
		Model    string ` + "`" + `yaml:"model"` + "`" + `
		BaseURL  string ` + "`" + `yaml:"base_url"` + "`" + `
	} ` + "`" + `yaml:"llm"` + "`" + `

	Memory struct {
		Backend string ` + "`" + `yaml:"backend"` + "`" + `
	} ` + "`" + `yaml:"memory"` + "`" + `

	Governance struct {
		Enable   bool     ` + "`" + `yaml:"enable"` + "`" + `
		Policies []string ` + "`" + `yaml:"policies"` + "`" + `
	} ` + "`" + `yaml:"governance"` + "`" + `

	Telemetry struct {
		Exporter    string ` + "`" + `yaml:"exporter"` + "`" + `
		Endpoint    string ` + "`" + `yaml:"endpoint"` + "`" + `
		ServiceName string ` + "`" + `yaml:"service_name"` + "`" + `
	} ` + "`" + `yaml:"telemetry"` + "`" + `
{{if .EnableMCP}}
	MCP struct {
		Enable  bool        ` + "`" + `yaml:"enable"` + "`" + `
		Servers []MCPServer ` + "`" + `yaml:"servers"` + "`" + `
	} ` + "`" + `yaml:"mcp"` + "`" + `
{{end}}
{{if .EnableA2A}}
	A2A struct {
		Enable     bool   ` + "`" + `yaml:"enable"` + "`" + `
		ListenAddr string ` + "`" + `yaml:"listen_addr"` + "`" + `
	} ` + "`" + `yaml:"a2a"` + "`" + `
{{end}}
}
{{if .EnableMCP}}
type MCPServer struct {
	Name    string   ` + "`" + `yaml:"name"` + "`" + `
	Command []string ` + "`" + `yaml:"command"` + "`" + `
}
{{end}}

func Load(basePath string, env string) (*Config, error) {
	cfg := &Config{}

	if err := loadYAML(basePath, cfg); err != nil {
		return nil, fmt.Errorf("loading base config: %w", err)
	}

	if env != "" {
		dir := filepath.Dir(basePath)
		envPath := filepath.Join(dir, fmt.Sprintf("config.%s.yaml", env))
		if _, err := os.Stat(envPath); err == nil {
			if err := loadYAML(envPath, cfg); err != nil {
				return nil, fmt.Errorf("loading %s config: %w", env, err)
			}
		}
	}

	return cfg, nil
}

func loadYAML(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, cfg)
}
`

const otelTemplate = `// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"context"

	"{{.Module}}/internal/config"
	"github.com/jllopis/kairos/pkg/telemetry"
)

func Init(ctx context.Context, cfg *config.Config) (shutdown func(context.Context) error, err error) {
	teleCfg := telemetry.Config{
		Exporter:     cfg.Telemetry.Exporter,
		OTLPEndpoint: cfg.Telemetry.Endpoint,
	}

	return telemetry.InitWithConfig(cfg.Telemetry.ServiceName, "1.0.0", teleCfg)
}
`
