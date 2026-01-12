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
