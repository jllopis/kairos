# Playbook 12 - Resilience

Goal: add retries, timeouts, circuit breakers, and fallbacks.

Incremental reuse:

- Add `internal/resilience` for retry/timeout/CB wrappers.

What to implement:

- Wrap external calls with `resilience.RetryConfig`.
- Use `resilience.NewCircuitBreaker` around DB or connector calls.
- Apply `resilience.WithTimeout` to tool calls and LLM calls.
- Use `resilience.WithFallback` for degraded responses.
- Return `errors.KairosError` with codes for observability.
- Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:

- Simulate failures and see retry/backoff.
- Circuit breaker opens after repeated failures.

Manual tests:

- Force a connector error and observe retries.
- Trigger enough failures to open the breaker.

Expected behavior:

- Retries back off and stop after max attempts.
- Circuit breaker blocks calls while open.

Checklist:

- [ ] Retry config is used for external calls.
- [ ] Circuit breaker state is visible in logs.
- [ ] Fallback returns a degraded response.

References:

- `examples/10-resilience-patterns`
- `pkg/resilience`
- `pkg/errors`
