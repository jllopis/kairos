# 01 - Hello Agent

El ejemplo más simple de Kairos: un agente que responde "Hello".

## Qué aprenderás

- Crear un agente con configuración mínima
- Usar `MockProvider` para desarrollo sin LLM real
- Ejecutar el ciclo básico de un agente

## Ejecutar

```bash
go run .
```

## Código clave

```go
// Crear agente con mock (sin LLM real)
llmProvider := llm.NewMockProvider()
ag, _ := agent.New("hello-agent", llmProvider,
    agent.WithRole("Greeter"),
)

// Ejecutar
result, _ := ag.Run(ctx, "Saluda")
```

## Siguiente paso

→ [02-basic-agent](../02-basic-agent/) para configuración y telemetría
