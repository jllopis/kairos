# Internal Agent (playbook)

Goal: help builders configure the SkyGuide specialized agent.

## What to implement

- A builder function that wires Role, Model, and common options.

## Suggested API

```go
package agent

import (
    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/llm"
)

// NewSkyGuide returns a pre-configured agent with SkyGuide roles
func NewSkyGuide(p llm.Provider, model string) (*agent.Agent, error) {
    return agent.New(
        agent.WithRole("You are SkyGuide..."),
        agent.WithProvider(p),
        agent.WithModel(model),
    )
}
```
