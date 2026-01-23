# Playbook 14 - Providers and Streaming

Goal: switch LLM providers and demonstrate streaming output.

Incremental reuse:

- Extend `internal/providers` with streaming support.

What to implement:

- CLI flag or config for provider selection.
- Instantiate provider:
  - `providers/openai`, `providers/gemini`
  - `llm.NewOllama`
  - `llm.MockProvider` for mock
- If provider implements `llm.StreamingProvider`, use `ChatStream`.
- Keep a mock path for offline runs.
- Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:

- Streaming prints partial output with supported providers.
- Non-streaming fallback works with mock provider.

Manual tests:

- "Write a short haiku about Go."

Expected behavior:

- Streaming prints output incrementally (when supported).
- Provider selection matches config/flags.

Checklist:

- [ ] Supports mock, ollama, openai, gemini.
- [ ] Non-streaming path still works.

References:

- `examples/16-providers`
- `examples/18-streaming`
- `pkg/llm`
