# Playbook 13 - Observability

Goal: add traces, metrics, and structured logs.

Incremental reuse:

- Extend `internal/observability` with custom spans/attributes.

What to implement:

- Init telemetry with stdout or OTLP exporter.
- Add custom spans around DB/connector/A2A calls.
- Use `telemetry.RecordError` for errors.
- Add attributes like run_id, session_id, tool_name, agent_id.
- Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:

- Stdout traces show custom spans.
- OTLP export works with a local backend if configured.

Manual tests:

- Run one request and inspect trace output.

Expected behavior:

- Spans include agent/tool identifiers and errors.

Checklist:

- [ ] Traces include run_id and session_id.
- [ ] Errors are recorded via `telemetry.RecordError`.

References:

- `examples/11-observability`
- `pkg/telemetry`
