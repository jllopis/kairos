# Bindings HTTP+JSON y JSON-RPC

Kairos implementa dos bindings principales para A2A. Ambos soportan streaming
vía SSE y publican el Agent Card en la ruta estándar cuando aplica.

## HTTP+JSON

Usa endpoints dedicados por operación y expone Agent Cards en
`/.well-known/agent-card.json`.

Ejemplo mínimo (servidor):

```go
handler := &server.SimpleHandler{
  Store:    server.NewMemoryTaskStore(),
  Executor: myExecutor{},
  Card:     myAgentCard(),
  PushCfgs: server.NewMemoryPushConfigStore(),
  ApprovalStore: server.NewMemoryApprovalStore(),
}

srv := httpjson.New(handler)
mux := http.NewServeMux()
mux.Handle("/", srv)
mux.Handle(agentcard.WellKnownPath, agentcard.PublishHandler(handler.AgentCard()))
```

Llamada de ejemplo:

```bash
curl -s http://localhost:8080/message:send \
  -H 'Content-Type: application/json' \
  -d '{
    "request": {
      "role": "ROLE_USER",
      "parts": [{ "text": "ping" }]
    }
  }'
```

## JSON-RPC

Usa un único endpoint con métodos A2A, pensado para integraciones más simples.

Ejemplo mínimo (servidor):

```go
handler := &server.SimpleHandler{
  Store:    server.NewMemoryTaskStore(),
  Executor: myExecutor{},
  Card:     myAgentCard(),
  PushCfgs: server.NewMemoryPushConfigStore(),
  ApprovalStore: server.NewMemoryApprovalStore(),
}

srv := jsonrpc.New(handler)
```

Llamada de ejemplo:

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
        "parts": [{ "text": "ping" }]
      }
    }
  }'
```

## Aprobaciones (HITL)

Cuando una política requiere aprobación, las tareas pueden entrar en estado
`INPUT_REQUIRED` y exponer un `approval_id`. Kairos permite listar, aprobar o
rechazar esas solicitudes desde la capa A2A.
