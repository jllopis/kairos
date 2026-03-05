# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0

.PHONY: build test vet lint clean tidy

## build: compile all packages and the kairos CLI
build:
	go build ./...

## test: run all tests with race detector
test:
	go test -race -count=1 ./...

## vet: run go vet on all packages
vet:
	go vet ./...

## lint: run golangci-lint (install: https://golangci-lint.run/welcome/install)
lint:
	golangci-lint run ./...

## tidy: tidy and verify module dependencies
tidy:
	go mod tidy
	go mod verify

## clean: remove build artifacts
clean:
	rm -rf bin/ dist/

## help: show this help message
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
