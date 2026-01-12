# Governance: Server-Side Enforcement and Human-in-the-Loop (HITL)

This note summarizes the current server-side governance enforcement and the
human-in-the-loop (HITL) approval flow.

## Goals
- Enforce `governance.PolicyEngine` decisions on A2A server handlers.
- Provide an optional HITL approval flow for sensitive actions.
- Preserve trace context and auditability across the decision path.

## Non-goals (for now)
- Full authN/authZ implementation (OIDC/mTLS).
- UI/CLI workflows for approvals (tracked in UI/CLI phase).

## Server-side A2A enforcement
Where evaluation happens:
- `pkg/a2a/server.SimpleHandler.SendMessage`
- `pkg/a2a/server.SimpleHandler.SendStreamingMessage`

Flow (high-level):
1) Extract identity metadata from the incoming request (agent name, caller, tenant).
2) Build a `governance.Action` with `Type: ActionAgent` and a stable `Name`
   (defaults to the handler's AgentCard name; falls back to `a2a-handler`).
3) Evaluate the policy before creating/executing a task.
4) If denied, return a task status `TASK_STATE_REJECTED` with a reason.

Notes:
- Metadata fields supported in request `metadata`: `caller`, `agent`, `tenant`.
- The same flow applies to HTTP+JSON/JSON-RPC handlers via the shared handler.

## HITL approval flow
Policy decisions support a pending state and `effect: "pending"` triggers an
approval request via the configured store.

Flow (high-level):
1) Policy engine evaluates the action. If `pending`, the handler requests approval.
2) The handler stores an `ApprovalRecord` in the configured `ApprovalStore`.
3) The handler returns `TASK_STATE_INPUT_REQUIRED` and includes metadata:
   `approval_id` and `approval_expires_at` (if a timeout is configured).
4) Operators call approve/reject endpoints. Approval executes the task.
5) Expired approvals are rejected by the sweeper or on access.

Storage and expiry:
- `ApprovalStore` supports in-memory and SQLite backends.
- `SimpleHandler.ApprovalTimeout` defines expiry; `ExpireApprovals` sweeps.
- Config: `governance.approval_timeout_seconds`,
  `runtime.approval_sweep_interval_seconds`, `runtime.approval_sweep_timeout_seconds`.

Operational expectations:
- Approvals are auditable and trace-linked to the original request.
- UI/CLI workflows for approvals are tracked in the UI/CLI phase.
