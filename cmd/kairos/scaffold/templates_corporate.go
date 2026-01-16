// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package scaffold

// Corporate template: GitHub Actions CI/CD
// GitHub Actions ${{ }} syntax is escaped using Go template literal syntax
var githubActionsTemplate = `# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

name: CI/CD

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main]

env:
  GO_VERSION: '1.22'
  REGISTRY: ghcr.io
  IMAGE_NAME: ${GITHUB_REPOSITORY}

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${GO_VERSION}

      - name: Cache Go modules
        uses: actions/cache@v4
        with:
          path: ~/go/pkg/mod
          key: ${RUNNER_OS}-go-${GITHUB_SHA}
          restore-keys: |
            ${RUNNER_OS}-go-

      - name: Download dependencies
        run: go mod download

      - name: Run tests
        run: go test -v -race -coverprofile=coverage.out ./...

      - name: Upload coverage
        uses: codecov/codecov-action@v4
        with:
          files: ./coverage.out
          fail_ci_if_error: false

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${GO_VERSION}

      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  build:
    runs-on: ubuntu-latest
    needs: [test, lint]
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${GO_VERSION}

      - name: Build
        run: go build -v ./...

  docker:
    runs-on: ubuntu-latest
    needs: [build]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    permissions:
      contents: read
      packages: write

    steps:
      - uses: actions/checkout@v4

      - name: Log in to Container Registry
        uses: docker/login-action@v3
        with:
          registry: ${REGISTRY}
          username: ${GITHUB_ACTOR}
          password: ${GITHUB_TOKEN}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${REGISTRY}/${IMAGE_NAME}
          tags: |
            type=sha,prefix=
            type=raw,value=latest

      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: ${STEPS_META_OUTPUTS_TAGS}
          labels: ${STEPS_META_OUTPUTS_LABELS}
`

// Corporate template: Dockerfile
var dockerfileTemplate = `# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/agent ./cmd/agent

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

# Copy binary from builder
COPY --from=builder /app/agent /app/agent

# Copy config files
COPY --from=builder /app/config /app/config

# Expose ports
# - 8080: HTTP API (if enabled)
# - 9090: Metrics endpoint
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/healthz || exit 1

# Default command
ENTRYPOINT ["/app/agent"]
CMD ["--config", "/app/config/config.yaml", "--profile", "prod"]
`

// Corporate template: Docker Compose for local development
var dockerComposeTemplate = `# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

version: '3.8'

services:
  agent:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - KAIROS_LLM_BASE_URL=http://ollama:11434
      - KAIROS_TELEMETRY_OTLP_ENDPOINT=otel-collector:4317
    depends_on:
      - ollama
      - otel-collector
    volumes:
      - ./config:/app/config:ro
    command: ["--config", "/app/config/config.yaml", "--profile", "dev"]

  ollama:
    image: ollama/ollama:latest
    ports:
      - "11434:11434"
    volumes:
      - ollama_data:/root/.ollama

  # OpenTelemetry Collector
  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    command: ["--config=/etc/otel-collector-config.yaml"]
    volumes:
      - ./deploy/otel-collector-config.yaml:/etc/otel-collector-config.yaml:ro
    ports:
      - "4317:4317"   # OTLP gRPC
      - "4318:4318"   # OTLP HTTP
      - "8888:8888"   # Prometheus metrics

  # Prometheus for metrics
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9091:9090"
    volumes:
      - ./deploy/prometheus.yaml:/etc/prometheus/prometheus.yml:ro
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  # Grafana for dashboards
  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
      - GF_USERS_ALLOW_SIGN_UP=false
    volumes:
      - grafana_data:/var/lib/grafana
      - ./deploy/grafana/provisioning:/etc/grafana/provisioning:ro

  # Jaeger for traces
  jaeger:
    image: jaegertracing/all-in-one:latest
    ports:
      - "16686:16686"  # Jaeger UI
      - "14268:14268"  # Collector HTTP
    environment:
      - COLLECTOR_OTLP_ENABLED=true

volumes:
  ollama_data:
  prometheus_data:
  grafana_data:
`

// Corporate template: OTEL Collector config
var otelCollectorConfigTemplate = `# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 1s
    send_batch_size: 1024

  memory_limiter:
    check_interval: 1s
    limit_mib: 512
    spike_limit_mib: 128

exporters:
  # Export to Jaeger for traces
  otlp/jaeger:
    endpoint: jaeger:4317
    tls:
      insecure: true

  # Export to Prometheus for metrics
  prometheus:
    endpoint: 0.0.0.0:8888
    namespace: kairos

  # Debug logging (disable in production)
  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlp/jaeger, debug]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [prometheus, debug]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [debug]
`

// Corporate template: Prometheus config
var prometheusConfigTemplate = `# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  # Scrape the agent directly
  - job_name: 'kairos-agent'
    static_configs:
      - targets: ['agent:9090']

  # Scrape OTEL Collector metrics
  - job_name: 'otel-collector'
    static_configs:
      - targets: ['otel-collector:8888']

  # Scrape Prometheus itself
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
`

// Corporate template: .dockerignore
var dockerignoreTemplate = `# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

# Git
.git
.gitignore

# IDE
.idea
.vscode
*.swp
*.swo

# Build artifacts
bin/
dist/
*.exe
*.dll
*.so
*.dylib

# Test artifacts
coverage.out
*.test

# OS files
.DS_Store
Thumbs.db

# Documentation
docs/
*.md
!README.md

# Development configs
config/*.dev.yaml
docker-compose*.yaml

# Secrets (never include)
.env
*.pem
*.key
secrets/
`

// Corporate template: golangci-lint config
var golangciLintTemplate = `# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gofmt
    - goimports
    - misspell
    - unconvert
    - gocritic
    - revive

linters-settings:
  govet:
    check-shadowing: true
  gofmt:
    simplify: true
  goimports:
    local-prefixes: {{.Module}}
  revive:
    rules:
      - name: blank-imports
      - name: context-as-argument
      - name: dot-imports
      - name: error-return
      - name: error-strings
      - name: exported
      - name: increment-decrement
      - name: var-declaration
      - name: package-comments
      - name: range
      - name: receiver-naming
      - name: time-naming
      - name: unexported-return
      - name: indent-error-flow
      - name: errorf

issues:
  exclude-use-default: false
  max-issues-per-linter: 0
  max-same-issues: 0
`
