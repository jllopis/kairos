# A2A Implementation Plan (Kairos)

This plan follows the normative A2A proto (`pkg/a2a/proto/a2a.proto`) and prioritizes gRPC streaming for the MVP.

## Goals
- Full protocol fidelity (types generated from proto).
- Interop-ready server + client with streaming.
- AgentCard discovery and capability negotiation.
- Trace continuity across agents.

## MVP Scope (gRPC binding)
1) **Types and version pinning**
   - Pin proto version and generate Go types.
   - Keep generated types isolated (e.g., `pkg/a2a/types`).
   - Generation script: `scripts/gen-a2a.sh`.
   - Requires googleapis protos (set `A2A_GOOGLEAPIS_DIR`).

2) **Server binding**
   - Implement A2AService gRPC: SendMessage, SendStreamingMessage, GetTask, ListTasks, CancelTask.
   - Map A2A Task/Message/Artifact to runtime state.
   - Streaming events honor backpressure and context cancellation.

3) **Client binding**
   - AgentCard discovery (well-known).
   - gRPC client for core methods + streaming.
   - Retry/timeout policy hooks + auth middleware stubs.

4) **AgentCard**
   - Generate card from agent capabilities and config.
   - Support `supportedInterfaces` for gRPC binding.

5) **Observability + tests**
   - Trace propagation across A2A boundaries.
   - Conformance tests (golden payloads, streaming order, cancel).

## Post-MVP
- HTTP+JSON and JSON-RPC bindings, con streaming vía SSE.
- Full authN/authZ integration (OIDC/mTLS).
- Push notification configuration + extended agent card.
