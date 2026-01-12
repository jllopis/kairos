# Instrucciones para ejecutar por separado cada uno de los agentes

Al ejecutar cada agente en su propia consola es más sencillo acceder a los logs de consola.

```bash
# Knowledge
KAIROS_TELEMETRY_EXPORTER=otlp \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
OLLAMA_URL=http://localhost:11434 \
KAIROS_LLM_PROVIDER=ollama \
KAIROS_LLM_MODEL=qwen2.5-coder:7b-instruct-q5_K_M \
go run ./cmd/knowledge --addr :9031 --qdrant localhost:6334 --embed-model nomic-embed-text \
--mcp-addr 127.0.0.1:9041 --card-addr 127.0.0.1:9141
```

```bash
# Spreadsheet
KAIROS_TELEMETRY_EXPORTER=otlp \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
OLLAMA_URL=http://localhost:11434 \
KAIROS_LLM_PROVIDER=ollama \
KAIROS_LLM_MODEL=qwen2.5-coder:7b-instruct-q5_K_M \
go run ./cmd/spreadsheet --addr :9032 --data ./data --qdrant localhost:6334 --embed-model nomic-embed-text \
--mcp-addr 127.0.0.1:9042 --card-addr 127.0.0.1:9142
```

```bash
# Orchestrator
KAIROS_TELEMETRY_EXPORTER=otlp \
KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317 \
KAIROS_TELEMETRY_OTLP_INSECURE=true \
OLLAMA_URL=http://localhost:11434 KAIROS_LLM_PROVIDER=ollama KAIROS_LLM_MODEL=qwen2.5-coder:7b-instruct-q5_K_M \
go run ./cmd/orchestrator --addr :9030 --knowledge localhost:9031 --spreadsheet localhost:9032 \
--qdrant localhost:6334 --embed-model nomic-embed-text --plan ./data/orchestrator_plan.yaml \
--knowledge-card-url http://127.0.0.1:9141 --spreadsheet-card-url http://127.0.0.1:9142 --card-addr 127.0.0.1:9140
```

# Cómo arrancar la UI con discovery demo

```bash
go run ./cmd/kairos --web --config docs/demo-settings.json --web-addr :8090
```
