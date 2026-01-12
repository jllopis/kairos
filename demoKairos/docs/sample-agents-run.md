# Instrucciones para ejecutar por separado cada uno de los agentes

Al ejecutar cada agente en su propia consola es mas sencillo acceder a los logs de consola.

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

# Como arrancar la UI con discovery demo

```bash
go run ./cmd/kairos --web --config docs/demo-settings.json --web-addr :8090
```

# Como iniciar la demo con autodiscovery

Se han anadido flags al launcher de la demo para configurar auto-register y se propagan como `KAIROS_DISCOVERY_*` para todos los agentes. Ahora se puede lanzar:

```bash
go run ./cmd/demo --registry-url http://localhost:9900 --auto-register --heartbeat 10
```

o por ENV:

```bash
KAIROS_DISCOVERY_REGISTRY_URL=http://localhost:9900 \
KAIROS_DISCOVERY_AUTO_REGISTER=true \
KAIROS_DISCOVERY_HEARTBEAT_SECONDS=10 \
go run ./cmd/demo
```
