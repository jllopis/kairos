# Playbook 10 - Governance

Goal: enforce allow/deny policies and tool filters.

Incremental reuse:

- Add `internal/governance` for policy engine and tool filter setup.

What to implement:

- Build a `governance.RuleSet` from config or inline rules.
- Attach policy engine with `agent.WithPolicyEngine`.
- Create a `governance.ToolFilter` (allowlist + denylist).
- Demonstrate a denied tool call with a clear reason.
- Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:

- A blocked tool call returns a policy error.
- Allowed tools still run.

Manual tests:

- "Use the forbidden tool to access data."

Expected behavior:

- Tool execution is denied with a policy reason.

Checklist:

- [ ] Deny rules take precedence.
- [ ] Allowlist restricts tools as expected.

References:

- `examples/08-governance-policies`
- `pkg/governance`
