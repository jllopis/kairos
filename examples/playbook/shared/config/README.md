# Internal config (playbook)

Goal: centralize config loading so every step reuses the same entrypoint.

## What to implement

- A helper that wraps `config.LoadWithCLI`.
- Support `--config`, `--profile`, and `--set` overrides.
- Expose a single config object for the rest of the internal packages.

## Suggested API

```go
package config

import (
    "github.com/jllopis/kairos/pkg/config"
)

// Load reads config from files and CLI flags
func Load(args []string) (*config.Config, error) {
    return config.LoadWithCLI(args)
}

// MustLoad is a helper for main that panics on error
func MustLoad() *config.Config {
    cfg, err := Load(os.Args[1:])
    if err != nil {
        panic(err)
    }
    return cfg
}
```

## Checklist

- [ ] Returns defaults when no config file exists.
- [ ] CLI overrides (`--set`) are applied correctly.
