# Playbook 03 - Tools

Goal: define a custom `core.Tool` and drive tool-calling end to end.

Incremental reuse:
- Add `internal/tools` and wire it through the shared agent builder.

What to implement:
- A tool type that implements:
  - `Name() string`
  - `Call(ctx, input any) (any, error)`
  - `ToolDefinition() llm.Tool`
- JSON Schema parameters in `ToolDefinition`.
- Use `llm.ScriptedMockProvider` to emit a tool call.
- Attach tools with `agent.WithTools`.
- (Optional) custom `core.EventEmitter` to observe tool usage.
- Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:
- The tool is invoked and its output is used in the final answer.

Manual tests:
- "Use the calculator tool to compute 2+2 and return the result."

Expected behavior:
- The agent triggers a tool call.
- The tool output appears in the final response.

Checklist:
- [ ] `ToolDefinition` includes a JSON schema.
- [ ] Tool call arguments are parsed correctly.
- [ ] Tool usage is observable (log/event/response).

References:
- `pkg/core/interfaces.go`
- `pkg/llm`
- `pkg/agent`
