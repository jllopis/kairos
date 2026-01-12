# Governance: Server-Side Enforcement and Human-in-the-Loop (Design Note)

This note captures the next governance gaps: enforcing policies on inbound A2A
requests and adding a human-in-the-loop (HITL) approval path.

## Goals
- Enforce `governance.PolicyEngine` decisions on A2A server handlers.
- Provide an optional HITL approval flow for sensitive actions.
- Preserve trace context and auditability across the decision path.

## Non-goals (for now)
- Full authN/authZ implementation (OIDC/mTLS).
- UI/CLI workflows for approvals (tracked in UI/CLI phase).

## Server-side A2A enforcement (proposal)
Where to evaluate:
- `pkg/a2a/server.SimpleHandler.SendMessage`
- `pkg/a2a/server.SimpleHandler.SendStreamingMessage`

Proposed flow (high-level):
1) Extract identity metadata from the incoming request (agent name, caller, org).
2) Build a `governance.Action` with `Type: ActionAgent` and a stable `Name`
   (e.g., local agent id, or `agent:<id>`).
3) Evaluate the policy before creating/executing a task.
4) If denied, return a gRPC error (e.g., `PermissionDenied`) with a reason.

Notes:
- Use `context.Context` to pass trace/identity metadata to the policy engine.
- The same approach applies to HTTP+JSON/JSON-RPC handlers via the shared handler.
- Add minimal audit logging for denies (policy rule id + reason).

TODOs:
- Decide the canonical `Action.Name` for server-side decisions.
- Specify which metadata is required for cross-tenant enforcement.

## HITL approval flow (proposal)
Current `Decision` only supports allow/deny. HITL needs a pending state.

Suggested direction:
- Extend the policy decision model to include a pending status (allow/deny/pending).
- Accept `effect: "pending"` in policy rules to trigger approvals.
- Add an approval hook to request and resolve approvals asynchronously.
- Store pending decisions with timeouts and correlation ids.
- Persist approvals with `ApprovalStore` (memory or SQLite).
- Use `SimpleHandler.ApprovalTimeout` to set expiry and `ExpireApprovals` to sweep.
- Config keys: `governance.approval_timeout_seconds` and `runtime.approval_sweep_interval_seconds`.

Minimal interface sketch (non-binding):
```
type ApprovalHook interface {
    Request(ctx context.Context, action governance.Action) (Decision, error)
}
```

Operational expectations:
- When a decision is pending, the server returns a "pending" task state and
  delays execution until approval is granted or expires.
- Approval actions should be auditable and trace-linked to the original request.

TODOs:
- Define storage backend and timeout semantics for approvals.
- Decide how "pending" maps to A2A task status and streaming events.
