# Playbook - Step by step example

Goal: build a full Kairos walkthrough from the simplest agent to a multi-agent orchestration with webhook + database + specialist agents.

## The Narrative: SkyGuide Travel Concierge

Throughout this playbook, you will build **SkyGuide**, an intelligent travel assistant.

- It starts as a simple "Hello" agent.
- It evolves into a service that remembers user preferences (Memory).
- It eventually coordinates complex flight bookings and hotel reservations (Orchestration).

## Rules

- No code is included here. Each step README tells you what to implement.
- Each step should be runnable on its own, but later steps build on prior ones.
- Build incrementally: reuse shared code from earlier steps instead of copying.
- Standardize provider selection: mock, ollama, openai, gemini.
- **Environment Variables**: Never commit API keys. Use environment variables (e.g., `OWM_API_KEY`) and reference them in your configuration or pass them at runtime.

## Incremental layout

- Create shared packages in `examples/playbook/internal` and reuse them.
- Each new step should only add what is new and wire it using those packages.

## Modules

### Module 1: Foundations

*Building the core of SkyGuide.*

1) [01-hello](file:///Users/jllopis/src/kairos/examples/playbook/01-hello/README.md) - Basic communication.
2) [02-config-telemetry](file:///Users/jllopis/src/kairos/examples/playbook/02-config-telemetry/README.md) - Professional setup.
3) [03-tools](file:///Users/jllopis/src/kairos/examples/playbook/03-tools/README.md) - Giving SkyGuide hands (check weather).
4) [04-a2a](file:///Users/jllopis/src/kairos/examples/playbook/04-a2a/README.md) - SkyGuide as a Service (Discovery & A2A).
5) [05-skills](file:///Users/jllopis/src/kairos/examples/playbook/05-skills/README.md) - Reusable specialized capabilities.

### Module 2: Advanced Logic

*Equipping SkyGuide for complex tasks.*
6) [06-memory](file:///Users/jllopis/src/kairos/examples/playbook/06-memory/README.md) - Remembering preferences.
7) [07-mcp](file:///Users/jllopis/src/kairos/examples/playbook/07-mcp/README.md) - Connecting to external platforms.
8) [08-planner](file:///Users/jllopis/src/kairos/examples/playbook/08-planner/README.md) - Deterministic booking flows.
9) [09-connectors](file:///Users/jllopis/src/kairos/examples/playbook/09-connectors/README.md) - Talking to Legacy APIs.
10) [10-governance](file:///Users/jllopis/src/kairos/examples/playbook/10-governance/README.md) - Booking policies and limits.

### Module 3: Operational Excellence

*Deploying a production-ready service.*
11) [11-guardrails](file:///Users/jllopis/src/kairos/examples/playbook/11-guardrails/README.md) - Sentiment and safety checks.
12) [12-resilience](file:///Users/jllopis/src/kairos/examples/playbook/12-resilience/README.md) - Handling API failures.
13) [13-observability](file:///Users/jllopis/src/kairos/examples/playbook/13-observability/README.md) - Advanced monitoring.
14) [14-providers-streaming](file:///Users/jllopis/src/kairos/examples/playbook/14-providers-streaming/README.md) - Real-time responses.
15) [15-testing](file:///Users/jllopis/src/kairos/examples/playbook/15-testing/README.md) - Automated validation.
16) [16-orchestration](file:///Users/jllopis/src/kairos/examples/playbook/16-orchestration/README.md) - Multi-agent coordination.

---
See [NARRATIVE.md](file:///Users/jllopis/src/kairos/examples/playbook/NARRATIVE.md) for a detailed evolution of SkyGuide.
