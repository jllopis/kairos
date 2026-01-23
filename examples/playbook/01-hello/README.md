# Playbook 01 - Hello SkyGuide

Goal: create the smallest working agent using a mock LLM provider.

## Why this step?

In Kairos, an **Agent** is more than just a prompt. it's a structural unit that combines a **Role**, a **Model**, and eventually **Tools** and **Memory**. Starting with a `MockProvider` allows you to test your agent's logic without worrying about API keys or network latency.

## SkyGuide Narrative

SkyGuide is born. At this stage, it's a simple entity that can acknowledge a user. We want to ensure the plumbing (Agent -> Provider -> Response) is working perfectly.

## Incremental reuse

- Keep this step minimal; step 02 will extract setup into `examples/playbook/internal`.

## What to implement

- `main.go` with `package main`.
- Create a mock provider (`llm.MockProvider` or `llm.ScriptedMockProvider`).
- `agent.New` with `agent.WithRole` ("You are SkyGuide, a travel assistant") and `agent.WithModel`.
- Call `Run(ctx, input)` and print the response.

## Suggested checks

- `go run .` prints the mocked response.
- No external services required.

## Manual tests

- "Hello"

## Expected behavior

- The mock provider returns the scripted response exactly as defined.

## Checklist

- [ ] Agent starts without config files.
- [ ] Response is printed once.

## References

- [01-hello-agent](file:///Users/jllopis/src/kairos/examples/01-hello-agent)
- `pkg/agent`
- `pkg/llm/mock.go`
