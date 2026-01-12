# A2A Implementation Plan (Kairos)

This plan follows the normative A2A proto (`pkg/a2a/proto/a2a.proto`) and prioritizes gRPC streaming for the MVP.

## Goals
- Full protocol fidelity (types generated from proto).
- Interop-ready server + client with streaming.
- AgentCard discovery and capability negotiation.
- Trace continuity across agents.

## MVP Scope (gRPC binding) - Status: Implemented
1) **Types and version pinning**
   - Pin proto version and generate Go types (`pkg/a2a/types`).
   - Generation script: `scripts/gen-a2a.sh`.
   - Requires googleapis protos (set `A2A_GOOGLEAPIS_DIR`).

2) **Server binding**
   - A2AService gRPC: SendMessage, SendStreamingMessage, GetTask, ListTasks, CancelTask.
   - Task/Message/Artifact mapping with streaming responses.
   - Streaming honors backpressure and context cancellation.

3) **Client binding**
   - AgentCard discovery (well-known).
   - gRPC client for core methods + streaming.
   - Retry/timeout policy hooks + auth middleware stubs.

4) **AgentCard**
   - Generated from agent capabilities and config.
   - `supportedInterfaces` include gRPC; extended card served via HTTP+JSON.

5) **Observability + tests**
   - Trace propagation across A2A boundaries.
   - Conformance tests (golden payloads, streaming order, cancel).
   - HTTP+JSON and JSON-RPC server bindings (SSE for streaming).

## Post-MVP
- Full authN/authZ integration (OIDC/mTLS) with server-side enforcement.
- Push notification configuration validation and client-side helpers.

## HTTP+JSON client helper sketch (implemented)

Target packages:
- `pkg/a2a/httpjson/client`
- `pkg/a2a/jsonrpc/client`

Core surface (HTTP+JSON):
- `New(baseURL string, opts ...Option) *Client`
- `SendMessage(ctx, *a2a.SendMessageRequest) (*a2a.SendMessageResponse, error)`
- `SendStreamingMessage(ctx, *a2a.SendMessageRequest) (<-chan *a2a.StreamResponse, error)`
- `GetTask(ctx, *a2a.GetTaskRequest) (*a2a.Task, error)`
- `ListTasks(ctx, *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error)`
- `CancelTask(ctx, *a2a.CancelTaskRequest) (*a2a.Task, error)`
- `SubscribeTask(ctx, *a2a.SubscribeToTaskRequest) (<-chan *a2a.StreamResponse, error)`
- `GetExtendedAgentCard(ctx, *a2a.GetExtendedAgentCardRequest) (*a2a.ExtendedAgentCard, error)`

Options:
- `WithHeaders(map[string]string)` for auth and tracing propagation.
- `WithHTTPClient(*http.Client)` for timeouts and transport tuning.

Notes:
- Streaming uses SSE to map server events to `StreamResponse`.
- Preserve OTel trace context by injecting headers from `context.Context`.
