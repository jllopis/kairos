# Templates Corporativos

La flag `--corporate` de `kairos init` genera infraestructura enterprise-ready adicional para proyectos que necesitan CI/CD, contenedorización y observabilidad desde el día uno.

## Uso

```bash
kairos init my-agent --module github.com/myorg/my-agent --corporate
```

## Archivos Generados

### CI/CD: `.github/workflows/ci.yaml`

Pipeline de GitHub Actions completo:

- **test**: Ejecuta tests con coverage
- **lint**: golangci-lint para calidad de código
- **build**: Verifica compilación
- **docker**: Build y push a GitHub Container Registry (solo en main)

```yaml
# Ejemplo de jobs generados
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go test -v -race -coverprofile=coverage.out ./...

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: golangci/golangci-lint-action@v6

  docker:
    needs: [test, lint, build]
    if: github.ref == 'refs/heads/main'
    # Push to ghcr.io
```

### Contenedorización: `Dockerfile`

Dockerfile multi-stage optimizado:

```dockerfile
# Build stage - compilación con todas las dependencias
FROM golang:1.22-alpine AS builder
# ...build con CGO_ENABLED=0 para binario estático

# Runtime stage - imagen mínima
FROM alpine:3.19
# Usuario non-root, health check incluido
```

Características:
- Build multi-stage (imagen final ~20MB)
- Usuario non-root por seguridad
- Health check integrado
- Soporte para config profiles

### Desarrollo Local: `docker-compose.yaml`

Stack completo para desarrollo:

```yaml
services:
  agent:        # Tu agente Kairos
  ollama:       # LLM local
  otel-collector:  # Collector de telemetría
  prometheus:   # Métricas
  grafana:      # Dashboards
  jaeger:       # Trazas distribuidas
```

**Puertos expuestos:**
- `8080`: API del agente
- `9090`: Métricas del agente
- `11434`: Ollama API
- `3000`: Grafana UI
- `16686`: Jaeger UI
- `9091`: Prometheus UI

### Observabilidad: `deploy/otel-collector-config.yaml`

Configuración del OpenTelemetry Collector:

```yaml
receivers:
  otlp:
    protocols:
      grpc: { endpoint: 0.0.0.0:4317 }
      http: { endpoint: 0.0.0.0:4318 }

exporters:
  otlp/jaeger:  # Trazas a Jaeger
  prometheus:   # Métricas a Prometheus
```

### Métricas: `deploy/prometheus.yaml`

Configuración de scraping:

```yaml
scrape_configs:
  - job_name: 'kairos-agent'
    targets: ['agent:9090']
  - job_name: 'otel-collector'
    targets: ['otel-collector:8888']
```

### Linting: `.golangci.yaml`

Configuración de linters recomendados:

- errcheck, govet, staticcheck
- gofmt, goimports
- misspell, gocritic, revive

## Flujo de Trabajo Recomendado

### Desarrollo Local

```bash
# Iniciar stack de observabilidad
docker-compose up -d ollama otel-collector prometheus grafana jaeger

# Ejecutar agente en modo dev
make run
# o
go run ./cmd/agent --config config/config.yaml --profile dev
```

### CI/CD

1. Push a rama feature → Tests + Lint
2. PR a main → Tests + Lint + Build
3. Merge a main → Tests + Lint + Build + Docker Push

### Producción

```bash
# Build de imagen
docker build -t my-agent:latest .

# Run con profile prod
docker run -e KAIROS_LLM_API_KEY=xxx my-agent:latest
```

## Personalización

### Añadir más linters

Edita `.golangci.yaml`:

```yaml
linters:
  enable:
    - errcheck
    - gosec      # Añadir security checks
    - bodyclose  # Añadir HTTP body close checks
```

### Cambiar registry de Docker

Edita `.github/workflows/ci.yaml`:

```yaml
env:
  REGISTRY: docker.io          # Docker Hub
  IMAGE_NAME: myorg/my-agent
```

### Añadir servicios al stack

Edita `docker-compose.yaml`:

```yaml
services:
  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

## Métricas Disponibles

Con el stack de observabilidad, puedes visualizar:

| Métrica | Descripción |
|---------|-------------|
| `kairos_agent_requests_total` | Total de requests al agente |
| `kairos_llm_latency_seconds` | Latencia de llamadas al LLM |
| `kairos_tool_calls_total` | Total de tool calls |
| `kairos_memory_operations_total` | Operaciones de memoria |

## Siguiente Paso

Ver `examples/12-production-layout/` para un ejemplo completo de estructura de producción.
