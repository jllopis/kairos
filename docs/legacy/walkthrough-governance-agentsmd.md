# Governance: AGENTS.md loader and policy engine

This walkthrough shows how to load AGENTS.md and use policy rules to gate tool calls.

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

`LoadAGENTS` walks upward from the provided directory until it finds `AGENTS.md`.

## Policy engine

Define rules and attach them to an agent:

```go
package main

import (
	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/governance"
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

func buildAgent(llmProvider any) (*agent.Agent, error) {
	return agent.New("demo-agent", llmProvider,
		agent.WithPolicyEngine(buildPolicy()),
	)
}
```

When a policy rule denies a tool call, the agent returns a policy observation and skips execution.

## Rule matching

- Rules are evaluated in order.
- `Name` uses simple glob matching (`*` wildcard).
- First match wins.
- If no rules match, the default decision is allow.
