# Guía de governance

Esta guía explica cómo activar AGENTS.md y aplicar políticas en Kairos.

## Carga de AGENTS.md

`LoadAGENTS` busca el archivo subiendo desde el directorio actual y devuelve su
contenido si existe.

```go
package main

import (
	"log"
	"os"

	"github.com/jllopis/kairos/pkg/governance"
)

func main() {
	cwd, _ := os.Getwd()
	doc, err := governance.LoadAGENTS(cwd)
	if err != nil {
		log.Fatalf("load agents: %v", err)
	}
	if doc == nil {
		log.Println("AGENTS.md not found")
		return
	}
	log.Printf("loaded %s", doc.Path)
	log.Printf("instructions: %s", doc.Raw)
}
```

## Política en un agente

Un `RuleSet` controla qué acciones se permiten o se bloquean. Se adjunta al
agente en la creación.

```go
package main

import (
	"context"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/governance"
	"github.com/jllopis/kairos/pkg/llm"
)

func buildPolicy() *governance.RuleSet {
	return governance.NewRuleSet([]governance.Rule{
		{
			ID:     "deny-secrets",
			Effect: "deny",
			Type:   governance.ActionTool,
			Name:   "secrets.*",
			Reason: "restricted tool",
		},
	})
}

func main() {
	llmProvider := llm.NewMock()
	agent, err := agent.New("demo-agent", llmProvider,
		agent.WithPolicyEngine(buildPolicy()),
	)
	if err != nil {
		panic(err)
	}
	_, _ = agent.Run(context.Background(), "hello")
}
```

## Políticas vía config

```json
{
  "governance": {
    "policies": [
      {
        "id": "deny-secrets",
        "effect": "deny",
        "type": "tool",
        "name": "secrets.*",
        "reason": "restricted tool"
      },
      {
        "id": "deny-remote",
        "effect": "deny",
        "type": "agent",
        "name": "external-*",
        "reason": "blocked agent"
      }
    ]
  }
}
```

Y luego:

```go
package main

import (
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/governance"
)

func buildPolicyFromConfig(path string) (*governance.RuleSet, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return governance.RuleSetFromConfig(cfg.Governance), nil
}
```

## Reglas y efectos

Las reglas se evalúan en orden, con glob matching en `Name` (por ejemplo,
`calc.*`, `db.query` o `*`). La primera coincidencia gana y, si no hay reglas,
la decisión por defecto es permitir. Puedes usar `effect: "pending"` para
disparar un flujo HITL.

## Ejemplo completo

Ver `examples/mcp-remote-policy-forbid` para un ejemplo ejecutable que bloquea
una tool MCP real mediante políticas.

## Notas de diseño

La parte de enforcement server-side y el flujo HITL están descritos en
`docs/governance-hitl.md`.

Para ejecución local con `kairos run`, puedes habilitar aprobaciones
interactivas con `--approval-mode ask`.

Para habilitar aprobaciones, configura `SimpleHandler.ApprovalStore` y usa
`effect: "pending"` en las reglas. Para expirar aprobaciones, ajusta
`SimpleHandler.ApprovalTimeout` y llama a `ExpireApprovals`.

Ejemplo de sweep desde el runtime:

```go
rt := runtime.NewLocalFromConfig(cfg.Runtime)
rt.AddApprovalExpirer(handler)
_ = rt.Start(context.Background())
```

Métricas emitidas por el sweeper:
`kairos.runtime.approval.sweep.count`,
`kairos.runtime.approval.sweep.error.count`,
`kairos.runtime.approval.expired.count`,
`kairos.runtime.approval.sweep.latency_ms`,
`kairos.runtime.approval.sweep.total_latency_ms`.

Config de ejemplo:

```json
{
  "runtime": {
    "approval_sweep_interval_seconds": 30,
    "approval_sweep_timeout_seconds": 5
  },
  "governance": {
    "approval_timeout_seconds": 300
  }
}
```

Wiring de ejemplo:

```go
if cfg.Governance.ApprovalTimeoutSeconds > 0 {
	handler.ApprovalTimeout = time.Duration(cfg.Governance.ApprovalTimeoutSeconds) * time.Second
}
if cfg.Runtime.ApprovalSweepIntervalSeconds > 0 {
	rt.SetApprovalSweepInterval(time.Duration(cfg.Runtime.ApprovalSweepIntervalSeconds) * time.Second)
}
if cfg.Runtime.ApprovalSweepTimeoutSeconds > 0 {
	rt.SetApprovalSweepTimeout(time.Duration(cfg.Runtime.ApprovalSweepTimeoutSeconds) * time.Second)
}
```
