# Governance Usage Guide

This guide shows how to enable AGENTS.md loading and policy enforcement in your own projects.

## Load AGENTS.md

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

## Add a policy engine to an agent

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

## Configure policies via config

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

Then load them:

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

## Policy deny example

See `examples/mcp-remote-policy-forbid` for a complete, runnable example that
denies a real MCP tool call via policy rules.

## Rule notes

- Rules are evaluated in order.
- `Name` uses glob matching (e.g., `calc.*`, `db.query`, `*`).
- First match wins.
- If no rules match, the default decision is allow.
- `effect: "pending"` can be used to trigger HITL approval flows.

## Design notes

- Server-side A2A enforcement and HITL proposals: `docs/governance-hitl.md`.
- To enable approvals, configure `SimpleHandler.ApprovalStore` and use `effect: "pending"` in rules.
- To expire approvals, set `SimpleHandler.ApprovalTimeout` and call `ExpireApprovals`.

Example: sweep expirations from the runtime loop:

```go
rt := runtime.NewLocalFromConfig(cfg.Runtime)
rt.AddApprovalExpirer(handler)
_ = rt.Start(context.Background())
```

Metrics emitted by the sweeper:
- `kairos.runtime.approval.sweep.count`
- `kairos.runtime.approval.sweep.error.count`
- `kairos.runtime.approval.expired.count`
- `kairos.runtime.approval.sweep.latency_ms`
- `kairos.runtime.approval.sweep.total_latency_ms`

Config example:

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

Wiring example:

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
