# Internal Providers (playbook)

Goal: modularize LLM provider creation.

## What to implement

- A factory function that reads `cfg.LLM.Provider`.
- Support for `mock`, `ollama`, `openai`, and `gemini`.

## Suggested API

```go
package providers

import (
    "github.com/jllopis/kairos/pkg/llm"
    "github.com/jllopis/kairos/pkg/config"
)

// New returns the LLM provider configured in the config object
func New(cfg *config.Config) (llm.Provider, error) {
    switch cfg.LLM.Provider {
    case "mock":
        return llm.MockProvider{}, nil
    case "ollama":
        // ... build ollama provider
    // ... other cases
    }
}
```
