# demoKairos

Demo multi-agente A2A con gRPC streaming, Qdrant y Ollama. Usa CSV locales como fuente de datos.

## Requisitos

- Qdrant en ejecución (gRPC default `localhost:6334`)
- Ollama en ejecución (default `http://localhost:11434`)
- Modelo de embeddings disponible (`nomic-embed-text` por defecto)

## Ejecutar agentes

Desde `demoKairos/`:

```bash
# Agente Knowledge (RAG)
go run ./cmd/knowledge --addr :9031 --qdrant localhost:6334 --embed-model nomic-embed-text

# Agente Spreadsheet (CSV)
go run ./cmd/spreadsheet --addr :9032 --data ./data

# Orchestrator

go run ./cmd/orchestrator --addr :9030 --knowledge localhost:9031 --spreadsheet localhost:9032 \
  --qdrant localhost:6334 --embed-model nomic-embed-text
```

## Probar con cliente gRPC

```bash
# Ventas Q4 por region

go run ./cmd/client --addr localhost:9030 --q "Cual fue el total de ventas en Q4 por region?"

# Top 10 productos por margen vs trimestre anterior

go run ./cmd/client --addr localhost:9030 --q "Dame el top 10 de productos por margen y comparalo con el trimestre anterior"

# Anomalias en Gastos

go run ./cmd/client --addr localhost:9030 --q "Que anomalias hay en la hoja Gastos este mes?"
```

## Scripts utiles

```bash
# Levantar todos los agentes
./scripts/run-demo.sh

# Healthcheck rapido de puertos gRPC
./scripts/healthcheck.sh

# Ejecutar una consulta de ejemplo
./scripts/run-query.sh "Cual fue el total de ventas en Q4 por region?"

# Ejecutar las tres consultas y guardar salida en ./outputs/<timestamp>
./scripts/run-sample-queries.sh
```

El script genera `outputs/summary.md` con timestamps y enlaces a cada salida.

El script de arranque incluye comprobaciones basicas para Qdrant (gRPC) y Ollama (HTTP) y mostrara avisos si no estan disponibles.

## Streaming semantico

El orchestrator emite eventos con `metadata.event_type` en `TaskStatusUpdateEvent`:

- `thinking`
- `retrieval.started` / `retrieval.done`
- `tool.started` / `tool.done`
- `response.final` (fin de stream)

Los mensajes incrementales se envian como `StreamResponse_Msg`.
