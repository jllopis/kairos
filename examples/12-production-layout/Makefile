.PHONY: run run-dev run-prod build test tidy

# Default: run with base config
run:
	go run ./cmd/agent --config ./config/config.yaml

# Development: use mock LLM, debug logging
run-dev:
	go run ./cmd/agent --config ./config/config.yaml --env dev

# Production: use real LLM, OTLP export
run-prod:
	go run ./cmd/agent --config ./config/config.yaml --env prod

build:
	go build -o bin/agent ./cmd/agent

test:
	go test ./...

tidy:
	go mod tidy
