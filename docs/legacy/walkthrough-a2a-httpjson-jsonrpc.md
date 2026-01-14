# A2A HTTP+JSON and JSON-RPC Bindings

This walkthrough shows how to expose an A2A handler over HTTP+JSON and JSON-RPC,
including streaming via SSE.

## HTTP+JSON binding

Minimal server setup:

```go
package main

import (
	"log"
	"net/http"

	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/httpjson"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
)

func main() {
	handler := &server.SimpleHandler{
		Store:    server.NewMemoryTaskStore(),
		Executor: myExecutor{}, // implements server.Executor
		Card:     myAgentCard(),
		PushCfgs: server.NewMemoryPushConfigStore(),
		ApprovalStore: server.NewMemoryApprovalStore(),
	}

	srv := httpjson.New(handler)
	mux := http.NewServeMux()
	mux.Handle("/", srv)
	mux.Handle(agentcard.WellKnownPath, agentcard.PublishHandler(handler.AgentCard()))

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func myAgentCard() *a2av1.AgentCard {
	streaming := true
	return &a2av1.AgentCard{
		ProtocolVersion: "v1",
		Name:            "Example Agent",
		Capabilities:    &a2av1.AgentCapabilities{Streaming: &streaming},
		SupportedInterfaces: []*a2av1.AgentInterface{
			{Url: "http://localhost:8080", ProtocolBinding: "http+json"},
		},
	}
}
```

Send a message:

```bash
curl -s http://localhost:8080/message:send \
  -H 'Content-Type: application/json' \
  -d '{
    "request": {
      "role": "ROLE_USER",
      "parts": [{ "text": "ping" }],
      "contextId": "ctx-1",
      "taskId": "task-1"
    }
  }'
```

Stream a message (SSE):

```bash
curl -N http://localhost:8080/message:stream \
  -H 'Content-Type: application/json' \
  -d '{
    "request": {
      "role": "ROLE_USER",
      "parts": [{ "text": "ping" }],
      "contextId": "ctx-1",
      "taskId": "task-1"
    }
  }'
```

SSE events contain JSON-encoded `StreamResponse` values:

```
data: {"statusUpdate":{"status":{"state":"TASK_STATE_WORKING"}}}
```

### Approvals (HITL)

If policy evaluation returns a pending decision, the task transitions to
`TASK_STATE_INPUT_REQUIRED` and includes an `approval_id` in the status message
metadata (plus `approval_expires_at` if a timeout is configured). You can list
approvals or approve/reject them:

```bash
curl -s http://localhost:8080/approvals?status=pending
curl -s http://localhost:8080/approvals?status=pending&expiresBefore=1735689600000
```

```bash
curl -s http://localhost:8080/approvals/APPROVAL_ID:approve \
  -H 'Content-Type: application/json' \
  -d '{"reason":"approved by operator"}'
```

```bash
curl -s http://localhost:8080/approvals/APPROVAL_ID:reject \
  -H 'Content-Type: application/json' \
  -d '{"reason":"blocked by policy"}'
```

## JSON-RPC binding

Minimal server setup:

```go
package main

import (
	"log"
	"net/http"

	"github.com/jllopis/kairos/pkg/a2a/jsonrpc"
	"github.com/jllopis/kairos/pkg/a2a/server"
)

func main() {
	handler := &server.SimpleHandler{
		Store:    server.NewMemoryTaskStore(),
		Executor: myExecutor{},
		Card:     myAgentCard(),
		PushCfgs: server.NewMemoryPushConfigStore(),
		ApprovalStore: server.NewMemoryApprovalStore(),
	}

	srv := jsonrpc.New(handler)
	log.Fatal(http.ListenAndServe(":8081", srv))
}
```

Send a message:

```bash
curl -s http://localhost:8081 \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "SendMessage",
    "params": {
      "request": {
        "role": "ROLE_USER",
        "parts": [{ "text": "ping" }],
        "contextId": "ctx-1",
        "taskId": "task-1"
      }
    }
  }'
```

List pending approvals:

```bash
curl -s http://localhost:8081 \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": "1",
    "method": "ListApprovals",
    "params": {"status": "pending"}
  }'
```

Approve an approval:

```bash
curl -s http://localhost:8081 \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": "2",
    "method": "ApproveApproval",
    "params": {"id": "APPROVAL_ID", "reason": "approved by operator"}
  }'
```

Stream a message (SSE):

```bash
curl -N http://localhost:8081 \
  -H 'Content-Type: application/json' \
  -d '{
    "jsonrpc": "2.0",
    "id": "stream-1",
    "method": "SendStreamingMessage",
    "params": {
      "request": {
        "role": "ROLE_USER",
        "parts": [{ "text": "ping" }],
        "contextId": "ctx-1",
        "taskId": "task-1"
      }
    }
  }'
```

Each SSE event is a JSON-RPC response that wraps a `StreamResponse`:

```
data: {"jsonrpc":"2.0","id":"stream-1","result":{"statusUpdate":{"status":{"state":"TASK_STATE_WORKING"}}}}
```

## Notes

- HTTP+JSON and JSON-RPC run on the same core handler interface as gRPC.
- Streaming uses SSE so responses remain visible while work happens.
- If your handler does not implement push notification methods, HTTP+JSON returns `501 Not Implemented`.
