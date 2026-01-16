# 12 - Production Layout

Estructura de proyecto recomendada para producción.

## Qué aprenderás

- Organización de código para proyectos reales
- Separación de concerns (config, app, observability)
- Configuración por entornos (dev/prod)
- Wiring explícito de componentes

## Estructura

```
12-production-layout/
├── cmd/
│   └── agent/
│       └── main.go           # Entrypoint mínimo
├── internal/
│   ├── app/
│   │   └── app.go            # Wiring de componentes
│   ├── config/
│   │   └── config.go         # Loader de configuración
│   └── observability/
│       └── otel.go           # Setup OTEL
├── config/
│   ├── config.yaml           # Config base
│   ├── config.dev.yaml       # Override desarrollo
│   └── config.prod.yaml      # Override producción
├── Makefile
└── README.md
```

## Ejecutar

```bash
# Desarrollo
make run

# Producción
make run-prod
```

## Principios de diseño

### 1. Entrypoint mínimo

`cmd/agent/main.go` solo hace:
- Parse de flags
- Carga de config
- Creación de app
- Manejo de señales

```go
func main() {
    cfg, _ := config.Load(*cfgPath)
    app, _ := app.New(cfg)
    app.Run(ctx)
}
```

### 2. Wiring explícito

`internal/app/app.go` conecta todos los componentes:
- LLM Provider
- Memory
- Governance
- Agent

Sin magia, sin inyección automática.

### 3. Config layering

```bash
# Base + override de entorno
config.yaml + config.dev.yaml  → desarrollo
config.yaml + config.prod.yaml → producción
```

### 4. Observabilidad desde el inicio

OTEL configurado antes de crear componentes.

## Este ejemplo es la base para `kairos init`

Cuando ejecutes `kairos init my-project`, generará esta misma estructura adaptada a tus necesidades.
