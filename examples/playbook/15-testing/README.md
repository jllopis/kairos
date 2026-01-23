# Playbook 14 - Testing

Goal: validate agent behavior with the testing harness.

Incremental reuse:
- Add `internal/testing` helpers for scenario setup and reuse.

What to implement:
- Use `testing.NewScenario` with `pkg/testing`.
- Drive the agent using `testing.ScenarioProvider`.
- Assert outputs, tool calls, events, and timing.
- Add at least one `*_test.go` file for the step.
 - Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:
- `go test ./...` passes.
- Scenario assertions fail when expected.

Manual tests:
- Break one expectation and confirm the test fails.

Expected behavior:
- Assertions report clear diffs for output/tool calls.

Checklist:
- [ ] At least one scenario validates tool calls.
- [ ] Setup/teardown hooks run.

References:
- `examples/15-testing`
- `pkg/testing`
