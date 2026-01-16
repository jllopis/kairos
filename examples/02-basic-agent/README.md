# 02 - Basic Agent

Agente con configuración externa y telemetría OpenTelemetry.

## Qué aprenderás

- Cargar configuración desde flags CLI o variables de entorno
- Inicializar telemetría OTEL (trazas y métricas)
- Usar un LLM real (Ollama) o mock
- El patrón de wiring de componentes en Kairos

## Requisitos

Para usar Ollama:
```bash
ollama serve
ollama pull llama3.1
```

## Ejecutar

Con mock (sin LLM real):
```bash
go run .
```

Con Ollama:
```bash
go run . --set llm.provider=ollama --set llm.model=llama3.1
```

O usando variables de entorno:
```bash
export KAIROS_LLM_PROVIDER=ollama
export KAIROS_LLM_MODEL=llama3.1
go run .
```

## Código clave

```go
// Configuración desde CLI o environment
cfg, err := config.LoadWithCLI(os.Args[1:])

// Telemetría OTEL
shutdown, err := telemetry.InitWithConfig("basic-agent", "v1.0", telemetry.Config{
    Exporter: cfg.Telemetry.Exporter,
})
defer shutdown(ctx)

// Crear provider según config
var provider llm.Provider
switch cfg.LLM.Provider {
case "ollama":
    provider = llm.NewOllama(cfg.LLM.BaseURL)
default:
    provider = &llm.MockProvider{Response: "Mock response"}
}

// Agente con config
ag, _ := agent.New("basic-agent", provider,
    agent.WithModel(cfg.LLM.Model),
)
```

## Siguiente paso

→ [03-memory-agent](../03-memory-agent/) para memoria semántica
