# Guía de desarrollo

<!--
Copyright 2026 © The Kairos Authors
SPDX-License-Identifier: Apache-2.0
-->

Esta guía cubre todo lo necesario para contribuir al desarrollo de Kairos:
entorno, herramientas, flujo de trabajo y convenciones.

## Requisitos previos

| Herramienta | Versión mínima | Instalación |
|-------------|---------------|-------------|
| Go | 1.25+ | [go.dev/dl](https://go.dev/dl/) |
| golangci-lint | v2.x | [golangci-lint.run/install](https://golangci-lint.run/welcome/install/) |
| Git | 2.x | Incluido en la mayoría de sistemas |
| Make | 3.x | Incluido en macOS/Linux |

Opcionales (para ejemplos y tests de integración):

| Herramienta | Uso |
|-------------|-----|
| Docker | Qdrant, backends OTEL, tests E2E |
| Node.js | Servidores MCP de ejemplo |
| protoc + protoc-gen-go | Regenerar tipos A2A |

## Configuración del entorno

### 1. Clonar el repositorio

```bash
git clone git@github.com:jllopis/kairos.git
cd kairos
```

### 2. Descargar dependencias

```bash
go mod download
```

### 3. Verificar que todo compila

```bash
make build
```

### 4. Ejecutar tests

```bash
make test
```

### 5. Ejecutar linter

```bash
make lint
```

## Makefile

El Makefile incluye todos los targets habituales:

| Target | Comando | Descripción |
|--------|---------|-------------|
| `make build` | `go build ./...` | Compila todos los paquetes y el CLI |
| `make test` | `go test -race -count=1 ./...` | Tests con race detector |
| `make vet` | `go vet ./...` | Análisis estático de Go |
| `make lint` | `golangci-lint run ./...` | Linter completo |
| `make tidy` | `go mod tidy && go mod verify` | Limpieza de dependencias |
| `make clean` | `rm -rf bin/ dist/` | Limpia artefactos de build |
| `make help` | — | Muestra la ayuda |

## Linting

### Configuración

El linter está configurado en `.golangci.yml` (formato v2). Incluye los linters
estándar de Go más:

| Linter | Qué detecta |
|--------|-------------|
| `bodyclose` | HTTP response bodies sin cerrar |
| `errorlint` | Uso incorrecto de `errors.Is` / `errors.As` / `%w` |
| `unconvert` | Conversiones de tipo innecesarias |
| `unparam` | Parámetros de función no usados |
| `misspell` | Errores ortográficos en inglés (locale US) |
| `nilerr` | Retorno de nil cuando se debería retornar error |

### Exclusiones

- **`examples/`**: excluido del análisis (código de demostración).
- **`defer .Close()`**: errcheck desactivado (patrón idiomático).
- **`_test.go`**: errcheck y errorlint relajados.
- **`fmt.Fprint*`**: errcheck desactivado en output de CLI.
- **`SA1019` (deprecated)**: APIs gRPC deprecated aceptadas temporalmente.
- **`QF*` / `ST*`**: sugerencias de estilo, no bloquean CI.

### Ejecución

```bash
# Lint completo
make lint

# Solo un paquete
golangci-lint run ./pkg/agent/...

# Con fix automático (formateadores)
golangci-lint run --fix ./...
```

## Tests

### Ejecutar todos los tests

```bash
make test
```

Esto ejecuta `go test -race -count=1 ./...`, que incluye:
- **Race detector**: detecta data races.
- **`-count=1`**: desactiva caché de tests para resultados fiables.

### Ejecutar tests de un paquete

```bash
go test -race ./pkg/agent/...
go test -race ./pkg/planner/...
```

### Ejecutar un test específico

```bash
go test -race -run TestAgentRun ./pkg/agent/
```

### Ver cobertura

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Cobertura por paquete

```bash
go test -cover ./pkg/...
```

### Tests de integración

Algunos tests requieren servicios externos (Qdrant, OTEL collector). Estos
usan build tags o variables de entorno para activarse:

```bash
# Test OTLP (requiere collector en localhost:4317)
KAIROS_OTLP_SMOKE_TEST=1 \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
go test ./pkg/telemetry -run TestOTLPSmoke -count=1
```

## CI (GitHub Actions)

El workflow de CI (`.github/workflows/ci.yml`) ejecuta en cada push y PR a
`main`/`master`:

### Job: build-and-test

1. Checkout
2. Setup Go 1.25
3. `go mod download`
4. `go vet ./...`
5. `go build ./...`
6. `go test -race -count=1 ./...`

### Job: lint

1. Checkout
2. Setup Go 1.25
3. `golangci-lint` via `golangci/golangci-lint-action@v6`

Ambos jobs corren en paralelo en `ubuntu-latest`.

## Convenciones de código

### Estilo general

- **Go fmt**: todo el código debe estar formateado con `gofmt`.
- **goimports**: imports organizados con prefijo local `github.com/jllopis/kairos`.
- **Context**: siempre como primer parámetro (`ctx context.Context`).
- **Errores**: usar `%w` para wrapping, `errors.Is` / `errors.As` para comparación.
- **Opciones**: patrón funcional (`WithXxx`) para configuración de structs.

### Licencia

Todos los ficheros de código fuente deben incluir el header de licencia:

```go
// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0
```

Para ficheros no-Go (YAML, Makefile, etc.):

```yaml
# Copyright 2026 © The Kairos Authors
# SPDX-License-Identifier: Apache-2.0
```

### Naming

| Elemento | Convención | Ejemplo |
|----------|------------|---------|
| Paquetes | minúsculas, singular | `agent`, `planner`, `memory` |
| Interfaces | sustantivo o adjetivo | `Provider`, `Recoverable` |
| Structs | PascalCase | `ChatRequest`, `KairosError` |
| Métodos | PascalCase, verbo | `Run`, `Store`, `Retrieve` |
| Constructores | `New` + tipo | `NewClient`, `NewResolver` |
| Options | `With` + campo | `WithModel`, `WithTimeout` |
| Constantes | PascalCase o `Code` prefijo | `CodeLLM`, `HealthStatusHealthy` |

### Gestión de goroutines

- Propagar `context.Context` en todas las goroutines.
- Evitar goroutines sin mecanismo de cancelación.
- Usar `sync.WaitGroup` o canales para coordinar finalización.
- Comprobar `ctx.Done()` en loops largos.

## Commits

Usamos [Conventional Commits](https://www.conventionalcommits.org/):

```
<tipo>: <descripción breve en imperativo>

[cuerpo opcional explicando el "por qué"]
```

### Tipos

| Tipo | Uso |
|------|-----|
| `feat` | Nueva funcionalidad |
| `fix` | Corrección de bug |
| `docs` | Solo documentación |
| `test` | Solo tests |
| `chore` | Mantenimiento, configuración |
| `ci` | Cambios en CI/CD |
| `refactor` | Refactoring sin cambio funcional |
| `perf` | Mejora de rendimiento |

### Ejemplos

```
feat: add GraphQL connector with schema introspection
fix: resolve context leak in a2a client streaming methods
docs: expand API.md with missing connector documentation
test: add integration tests for MCP pool health checks
chore: add golangci-lint v2 configuration
ci: add GitHub Actions workflow for build, test, and lint
```

## Flujo de contribución

### 1. Crear rama

```bash
git checkout -b feat/my-feature main
```

Naming de ramas:
- `feat/descripcion` — nueva funcionalidad
- `fix/descripcion` — corrección
- `docs/descripcion` — documentación
- `chore/descripcion` — mantenimiento

### 2. Desarrollar

- Escribir código + tests.
- Ejecutar `make test` y `make lint` antes de commitear.
- Commits pequeños y atómicos.

### 3. Verificar antes de push

```bash
make build && make test && make lint
```

### 4. Push y PR

```bash
git push -u origin feat/my-feature
```

Crear PR contra `main` con:
- Descripción clara del cambio.
- Referencia a issues si aplica.
- Tests que cubran el cambio.

### 5. Revisión y merge

- Mínimo 1 reviewer (2 para cambios críticos).
- CI debe pasar (build + test + lint).
- Merge y borrar rama.

## Estructura del proyecto

```
kairos/
├── .github/workflows/   # CI (GitHub Actions)
├── .golangci.yml         # Configuración de linter
├── Makefile              # Targets de build
├── AGENTS.md             # Instrucciones para agentes IA
├── go.mod / go.sum       # Dependencias
├── cmd/kairos/           # CLI principal
├── pkg/                  # Paquetes del framework
│   ├── agent/            # Agent runtime
│   ├── a2a/              # Protocolo A2A
│   ├── config/           # Tipos de configuración
│   ├── connectors/       # Conectores (OpenAPI, GraphQL, etc.)
│   ├── core/             # Interfaces compartidas
│   ├── discovery/        # Descubrimiento de agentes
│   ├── errors/           # Errores tipados
│   ├── governance/       # Motor de políticas
│   ├── guardrails/       # Filtros de seguridad
│   ├── llm/              # Interface LLM + Ollama
│   ├── mcp/              # Cliente/servidor MCP
│   ├── memory/           # Memoria (vector + conversación)
│   ├── planner/          # Planner DAG
│   ├── resilience/       # Retry, circuit breaker, etc.
│   ├── runtime/          # Runtime de orquestación
│   ├── skills/           # Cargador de AgentSkills
│   ├── telemetry/        # OpenTelemetry
│   └── testing/          # Helpers para tests
├── providers/            # LLM providers externos
│   ├── openai/
│   ├── anthropic/
│   ├── gemini/
│   └── qwen/
├── examples/             # 20 ejemplos progresivos
│   ├── 01-hello-agent/
│   ├── ...
│   └── 20-conversation-memory/
└── docs/                 # Documentación
```

## Recursos

- [Arquitectura](ARCHITECTURE.md)
- [API Reference](API.md)
- [Roadmap](ROADMAP.md)
- [Visión y Plan](VISION_AND_PLAN.md)
