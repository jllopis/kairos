// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package scaffold

// Config YAML templates

const configYAMLTemplate = `# {{.ProjectName}} configuration
# Archetype: {{.Archetype}}

app:
  name: "{{.ProjectName}}"
  log_level: "info"

llm:
  provider: "{{.LLMProvider}}"
  model: "llama3.1"
  base_url: "http://localhost:11434"

memory:
  backend: "inmemory"

governance:
{{- if eq .Archetype "policy-heavy"}}
  enable: true
  policies:
    - "tool:read_*"
    - "tool:search_*"
    # Deny by default - add allowed patterns above
{{- else}}
  enable: true
  policies:
    - "tool:*"  # Allow all tools by default
{{- end}}

telemetry:
  exporter: "stdout"
  endpoint: ""
  service_name: "{{.ProjectName}}"
{{if .EnableMCP}}
mcp:
  enable: true
  servers:
    # Example: filesystem server
    # - name: "filesystem"
    #   command: ["npx", "-y", "@modelcontextprotocol/server-filesystem", "."]
{{end}}
{{if .EnableA2A}}
a2a:
  enable: true
  listen_addr: "127.0.0.1:8080"
{{end}}
`

const configDevYAMLTemplate = `# Development overrides

app:
  log_level: "debug"

llm:
  provider: "mock"

telemetry:
  exporter: "stdout"
`

const configProdYAMLTemplate = `# Production overrides

app:
  log_level: "warn"

llm:
  provider: "ollama"

telemetry:
  exporter: "otlp"
  endpoint: "localhost:4317"
`
