<!--
Copyright 2026 © The Kairos Authors
SPDX-License-Identifier: Apache-2.0
-->

# Changelog

All notable changes to Kairos are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **CI**: GitHub Actions workflow for build, test (with race detector), and golangci-lint.
- **Build**: Makefile with `build`, `test`, `vet`, `lint`, `clean`, `tidy`, and `help` targets.
- **Lint**: golangci-lint v2 configuration (`.golangci.yml`) with tuned exclusion rules.
- **Docs**: Vision and plan document (`VISION_AND_PLAN.md`) with 6-phase improvement roadmap.
- **Docs**: Rewritten `index.md` with full navigation linking all 60+ documentation files.
- **Docs**: Rewritten `ARCHITECTURE.md` with 10 Mermaid diagrams (C4, ReAct/Planner flows, MCP/A2A sequences).
- **Docs**: Expanded `API.md` from 10 to 19 sections (added Connectors, Guardrails, Errors, Resilience, Telemetry, Events, Health, Discovery, Streaming).
- **Docs**: New `DEVELOPER_GUIDE.md` covering environment setup, lint, tests, CI, conventions, and contribution workflow.
- **Docs**: Expanded `Conceptos_AgentLoop.md` and `Conceptos_Memoria.md` with Mermaid diagrams and code examples.
- **Docs**: README for `examples/mcp-http-server`.

### Fixed

- Context leak in A2A client streaming methods (`a2a/client`).
- Resolved 103 golangci-lint issues across 33 source files (127 → 24 remaining, all accepted).
- Broken cross-references in `GUARDRAILS.md`, `CONVERSATION_MEMORY.md`, and `governance-usage.md`.
- Compiled binaries (`16-providers`, `18-streaming`) removed from git tracking with updated `.gitignore`.

## [0.2.5] - 2026-02-28

This is a major milestone release consolidating A2A protocol support, governance,
planner enhancements, discovery, connectors, LLM providers, and production hardening.

### Added

- **A2A Protocol**: Full Agent-to-Agent protocol implementation:
  - gRPC client/server scaffolding with protobuf type generation.
  - HTTP+JSON and JSON-RPC transport bindings.
  - Task store with CRUD, subscriptions, push config, and pagination.
  - AgentCard discovery and capability checks.
  - Client retries, timeouts, authentication, and trace context propagation.
  - Server conformance tests and golden test suites (100+ A2A test cases).
  - SQLite-backed task and push config stores.
- **Planner**: Explicit planner with branching, audit hooks, advanced conditions, and node ID handlers.
  - Planner audit store for execution tracing.
  - Default node aliases for legacy graph compatibility.
  - Runtime contract documentation and plan examples.
- **Governance**: Policy engine with config-driven policies, HITL approval workflows, and CLI overrides.
  - Config-driven governance policies with MCP deny example.
  - Runtime sweeper for expired approvals.
- **Discovery**: Provider architecture with registry support, well-known endpoints, and auto-registration.
- **LLM Providers**: OpenAI, Anthropic, Qwen, and Gemini providers (all implementing `Provider` and `StreamingProvider`).
  - Provider authentication and token usage example.
  - Streaming support for Qwen and Ollama providers.
- **Connectors**: OpenAPI, GraphQL (with introspection), gRPC, and SQL connectors.
- **Guardrails**: Security guardrails system (content filter, PII detection, prompt injection).
  - Guardrails runtime integration with planner telemetry.
- **Testing**: Agent testing framework with `NewScriptedMockProvider`.
- **MCP**: Shared connection pool for multi-agent scenarios.
- **Config**: Config layering with environment profiles and hot-reload via file watcher.
- **Scaffold**: Corporate templates for enterprise projects.
- **Memory**: Conversation memory with vector store integration (Ollama embeddings, Qdrant).
  - Hardened conversation storage and async tasks (KAIROS-002).
- **Core**: Task, role, and event types. OTLP authentication support. Skills as tools.
  - Tool definition enforcement and runtime hardening (KAIROS-001).
  - Trace-aware logging and web UI configuration.
- **Examples**: 20 numbered examples (01-hello-agent through 20-conversation-memory) plus playbook and mcp-http-server.
- **Docs**: Comprehensive documentation suite — error handling guides, integration guide, metrics export, CONNECTORS.md, provider architecture, observability guide, API reference, and ADR-0005.
- **CLI**: Advanced commands with config overrides and web streaming UI.

### Changed

- Reorganized examples from flat structure to numbered progression.
- Elevated A2A to Step 04 in playbook, renumbered subsequent steps.
- Removed demoKairos in favor of playbook-based workflow.
- Legacy Action fallback disabled by default in emergent planner.
- Default models optimized for cost/quality: `gpt-5-mini` (OpenAI), `gemini-3-flash-preview` (Gemini).

### Fixed

- Telemetry env key normalization and OTLP endpoint env override.
- Demo runner config args, expansion, and verbose flag handling.
- A2A delegation task ID handling.
- `.gitignore` corrected to only ignore root `kairos` binary.
- Streaming default model for Ollama provider.
- CancelTask made idempotent for terminal states.
- Negative page size and history length rejected in A2A ListTasks/GetTask.

## [0.2.4] - 2026-01-10

### Added

- Ollama as remote MCP agent provider in example.

## [0.2.3] - 2026-01-10

### Added

- Configurable OTLP export timeout.

## [0.2.2] - 2026-01-10

### Fixed

- OTLP endpoint env variable override now honored correctly.

## [0.2.1] - 2026-01-10

### Fixed

- Telemetry env key normalization for consistent configuration.

## [0.2.0] - 2026-01-10

### Added

- HTTP server run instructions for MCP streamable transport.

## [0.1.9] - 2026-01-10

### Added

- MCP streamable HTTP server example.

## [0.1.8] - 2026-01-10

### Added

- HTTP MCP transport walkthrough documentation.

## [0.1.7] - 2026-01-10

### Added

- Streamable HTTP transport for MCP protocol.

## [0.1.6] - 2026-01-10

### Added

- Telemetry environment variables documentation.

## [0.1.5] - 2026-01-10

### Added

- OTLP verification steps in documentation.

## [0.1.4] - 2026-01-10

### Changed

- Roadmap and user stories updated to reflect OTLP progress.

## [0.1.3] - 2026-01-10

### Added

- OTLP telemetry configuration example.

## [0.1.2] - 2026-01-10

### Added

- Configurable OTLP telemetry exporter with OpenTelemetry integration.

## [0.1.1] - 2026-01-10

### Added

- MCP client hardening with retries and caching.

## [0.1.0] - 2026-01-10

Initial release of the Kairos AI Agent Framework.

### Added

- Framework skeleton with core agent runtime.
- Configuration management and structured logging.
- MCP (Model Context Protocol) config loading and tool registration.
- Structured tool calling for LLM interactions.
- User stories and roadmap milestones.

[Unreleased]: https://github.com/jllopis/kairos/compare/v0.2.5...HEAD
[0.2.5]: https://github.com/jllopis/kairos/compare/v0.2.4...v0.2.5
[0.2.4]: https://github.com/jllopis/kairos/compare/v0.2.3...v0.2.4
[0.2.3]: https://github.com/jllopis/kairos/compare/v0.2.2...v0.2.3
[0.2.2]: https://github.com/jllopis/kairos/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/jllopis/kairos/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/jllopis/kairos/compare/v0.1.9...v0.2.0
[0.1.9]: https://github.com/jllopis/kairos/compare/v0.1.8...v0.1.9
[0.1.8]: https://github.com/jllopis/kairos/compare/v0.1.7...v0.1.8
[0.1.7]: https://github.com/jllopis/kairos/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/jllopis/kairos/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/jllopis/kairos/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/jllopis/kairos/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/jllopis/kairos/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/jllopis/kairos/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/jllopis/kairos/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/jllopis/kairos/releases/tag/v0.1.0
