# Kairos

Relación entre paquetes y cómo habilitan flujos multi‑agente.

**Idea central**
- `pkg/a2a` no es “el core lógico” del framework; es la **capa de interoperabilidad y transporte** que permite que agentes (propios o externos) se comuniquen con un contrato estándar.
- El **core de ejecución** está en `pkg/agent` (runtime del agente), `pkg/llm`, `pkg/mcp`, `pkg/memory`, y `pkg/planner`.  
- `pkg/a2a` es la “red”; `pkg/agent` es el “cerebro”.

---

## Relación entre paquetes (visión por capas)

**1) Runtime del agente**
- `pkg/agent`: ciclo de razonamiento, herramientas, memoria y ejecución de tareas. Es el núcleo que “decide” y “actúa”.
- `pkg/llm`: proveedor LLM (p. ej. Ollama) usado por `agent`.
- `pkg/mcp`: integración de tools/servicios externos; el agente puede llamar a herramientas declaradas vía MCP.
- `pkg/memory`: memoria y almacenamiento vectorial (Qdrant).
- `pkg/config`: configura todo lo anterior.
- `pkg/telemetry`: observabilidad (trazas/métricas).

**2) Orquestación y flujos**
- `pkg/planner`: define un plan (grafo) con nodos/edges y un executor.
- Los nodos del plan pueden **llamar agentes** (locales o remotos).

**3) Interoperabilidad**
- `pkg/a2a`: capa de red para **comunicación agent‑to‑agent** (SendMessage/SendStreamingMessage, AgentCard, Task, etc.).
- Expone agentes como servicios gRPC/HTTP bajo un estándar: descubrimiento, capacidades, streaming, etc.

---

## Cómo se forma un flujo multi‑agente

**Estructura base**:
1) Un agente se define con `pkg/agent` (roles, tools MCP, memoria, llm).
2) Si quieres que otros lo llamen, lo publicas vía `pkg/a2a` (gRPC + AgentCard).
3) Un orquestador (otro `agent` o un `planner`) decide a quién delegar.
4) La delegación ocurre por A2A, y cada agente responde con mensajes/tareas estandarizadas.
5) Observabilidad (`pkg/telemetry`) te da trazas y métricas coherentes entre agentes.

---

## Formas de colaboración entre agentes

**A) Orquestador central**
- Un agente “maestro” decide y delega.
- Ideal para flujos guiados y experiencia controlada.
- Implementación: `planner` + `agent` + `a2a`.

**B) Peer‑to‑peer**
- Cada agente puede llamar a otros directamente.
- Útil para redes descentralizadas o federación.
- Implementación: cada agente expone `a2a` + consume `a2a` de otros.

**C) Hub‑and‑spoke**
- Un “router” A2A decide destino en base a AgentCard/capacidades.
- Útil para discovery dinámico.

**D) Workflow explícito con `planner`**
- Un plan (YAML/JSON) define los pasos.
- Cada paso puede ser un agente A2A o un tool MCP.

---

## Cómo un desarrollador crea agentes autónomos con Kairos

**1) Define el agente**
- `pkg/agent`: role, model, tools MCP, memoria.

**2) Define tools**
- `pkg/mcp`: herramientas internas/externas con esquema claro.

**3) Añade memoria**
- `pkg/memory`: contexto persistente y RAG.

**4) Orquesta**
- `pkg/planner` para flujos deterministas o `agent` si es autónomo.

**5) Publica y conecta**
- `pkg/a2a`: expone el agente (server) o consume otros (client).

**6) Observabilidad**
- `pkg/telemetry`: trazas y métricas end‑to‑end.

---

## Ejemplo minimo (flujo completo, educativo)

Un developer crea un agente simple, lo publica por A2A y otro proceso lo invoca con `SendMessage`.

```go
// main.go
package main

import (
	"context"
	"log"
	"net"
	"net/http"

	"github.com/jllopis/kairos/pkg/a2a/agentcard"
	"github.com/jllopis/kairos/pkg/a2a/client"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/pkg/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// 0) MCP: tool local via Streamable HTTP (inicia antes del agente)
	mcpServer := mcp.NewServer("demo-mcp", "0.1.0")
	mcpServer.RegisterTool("hello", "Devuelve un saludo", nil, func(ctx context.Context, args map[string]interface{}) (*mcpgo.CallToolResult, error) {
		name, _ := args["name"].(string)
		if name == "" {
			name = "mundo"
		}
		return &mcpgo.CallToolResult{StructuredContent: map[string]interface{}{"text": "hola " + name}}, nil
	})
	httpSrv := mcpServer.StreamableHTTPServer()
	go func() { _ = httpSrv.Start("127.0.0.1:9901") }()
	mcpClient, _ := mcp.NewClientWithStreamableHTTP("http://127.0.0.1:9901/mcp")

	// 1) LLM + agente
	provider := llm.NewOllama("http://localhost:11434")
	a, _ := agent.New(
		"hello-agent",
		provider,
		agent.WithRole("Usa la tool hello para responder de forma corta y clara."),
		agent.WithModel("qwen2.5-coder:7b-instruct-q5_K_M"),
		agent.WithMCPClients(mcpClient),
	)

	// 2) AgentCard para discovery
	card := agentcard.Build(agentcard.Config{
		ProtocolVersion: "v1",
		Name:            "Hello Agent",
		Description:     "Agente de ejemplo.",
		Version:         "0.1.0",
		Capabilities: func() *a2av1.AgentCapabilities {
			streaming := true
			return &a2av1.AgentCapabilities{Streaming: &streaming}
		}(),
		SupportedInterfaces: []*a2av1.AgentInterface{
			{Url: "grpc://localhost:9035", ProtocolBinding: "grpc"},
		},
	})

	// 3) Exponer el agente por A2A
	handler := server.NewAgentHandler(a, server.WithAgentCard(card))
	svc := server.New(handler)
	grpcServer := grpc.NewServer()
	a2av1.RegisterA2AServiceServer(grpcServer, svc)

	mux := http.NewServeMux()
	mux.Handle(agentcard.WellKnownPath, agentcard.PublishHandler(card))
	go func() { _ = http.ListenAndServe("127.0.0.1:9135", mux) }()

	lis, _ := net.Listen("tcp", ":9035")
	log.Println("hello-agent listo en :9035")
	_ = grpcServer.Serve(lis)
}
```

Cliente A2A (otro proceso):

```go
conn, _ := grpc.Dial("localhost:9035", grpc.WithTransportCredentials(insecure.NewCredentials()))
cli := client.New(conn)
msg := &a2av1.Message{
	MessageId: "msg-1",
	ContextId: "ctx-1",
	Role:      a2av1.Role_ROLE_USER,
	Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "Hola Kairos"}}},
}
resp, _ := cli.SendMessage(context.Background(), &a2av1.SendMessageRequest{
	Request:       msg,
	Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
})
_ = resp
```

Este ejemplo muestra el ciclo completo: agent local -> exposicion A2A -> discovery -> llamada remota.

---

## Cómo encaja esto con la demo actual

- **Knowledge Agent**: `agent` + `memory` + Qdrant, expuesto por `a2a`.
- **Spreadsheet Agent**: `agent` + tools MCP (CSV), expuesto por `a2a`.
- **Orchestrator**: `planner` + `agent` y usa `a2a` para delegar.
- El **flujo completo** lo conectan `planner` (lógica), `a2a` (transporte), `agent` (ejecución) y `mcp` (tools).

---

## Mensaje de justificación simple

- `pkg/agent` es la ejecución local.
- `pkg/a2a` es la red que permite que varios agentes se coordinen como un sistema distribuido.
- `pkg/planner` convierte esa coordinación en flujos reproducibles.
- `pkg/mcp`, `pkg/memory`, `pkg/llm` son los subsistemas que dan capacidades al agente.
