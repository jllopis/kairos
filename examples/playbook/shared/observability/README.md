# Internal Observability (playbook)

Goal: handle OpenTelemetry lifecycle.

## What to implement

- Initialization of the OTLP or Stdout exporter.
- Clean shutdown function (returned during init).

## Suggested API

```go
package observability

import (
    "github.com/jllopis/kairos/pkg/telemetry"
    "github.com/jllopis/kairos/pkg/config"
)

// Init sets up the global tracer and returns a shutdown function
func Init(cfg *config.Config) (func(), error) {
    return telemetry.InitWithConfig(cfg)
}
```
