# Guía de configuración

La configuración de Kairos se puede definir por archivo, variables de entorno y
flags de CLI. La precedencia es la siguiente:

1) Valores por defecto
2) Archivo (`~/.kairos/settings.json`, `./.kairos/settings.json` o `XDG_CONFIG_HOME/kairos/settings.json`)
3) Variables de entorno (`KAIROS_*`)
4) Sobrescrituras de CLI (`--config`, `--set`)

## Configuración por archivo

Crea un `settings.json` en una de las rutas soportadas:

- `./.kairos/settings.json`
- `$HOME/.kairos/settings.json`
- `$XDG_CONFIG_HOME/kairos/settings.json`

Ejemplo (mínimo):

```json
{
  "llm": {
    "provider": "ollama",
    "model": "qwen2.5-coder:7b-instruct-q5_K_M"
  },
  "telemetry": {
    "exporter": "stdout"
  }
}
```

Ejemplo (con MCP y discovery):

```json
{
  "discovery": {
    "order": ["config", "well_known", "registry"],
    "registry_url": "http://localhost:9900",
    "registry_token": "token-if-needed",
    "auto_register": false,
    "heartbeat_seconds": 10
  },
  "mcp": {
    "servers": {
      "fetch": {
        "transport": "stdio",
        "command": "docker",
        "args": ["run", "-i", "--rm", "mcp/fetch"]
      }
    }
  },
  "agents": {
    "orchestrator": {
      "agent_card_url": "http://127.0.0.1:9140",
      "grpc_addr": "127.0.0.1:9030",
      "http_url": "http://127.0.0.1:8080",
      "labels": {"env": "local", "tier": "core"}
    }
  },
  "runtime": {
    "approval_sweep_interval_seconds": 30,
    "approval_sweep_timeout_seconds": 5
  },
  "governance": {
    "approval_timeout_seconds": 300
  }
}
```

## Variables de entorno

Las variables de entorno se mapean a claves de configuración así:

1) Prefijo: `KAIROS_`
2) Minúsculas
3) `_` se convierte en `.`

Ejemplos de variables:

```bash
KAIROS_LLM_PROVIDER=ollama
KAIROS_LLM_MODEL=qwen2.5-coder:7b-instruct-q5_K_M
KAIROS_TELEMETRY_EXPORTER=stdout
KAIROS_RUNTIME_APPROVAL_SWEEP_INTERVAL_SECONDS=30
KAIROS_GOVERNANCE_APPROVAL_TIMEOUT_SECONDS=300
```

## Sobrescrituras de CLI

El CLI soporta una ruta de config y sobrescrituras repetibles:

- `--config=/ruta/a/settings.json`
- `--set key=value`

Ejemplo de CLI:

```bash
go run ./examples/basic-agent --config=./.kairos/settings.json \
  --set llm.provider=ollama \
  --set telemetry.exporter=stdout
```

Valores JSON con `--set`:

```bash
go run ./examples/mcp-agent \
  --set mcp.servers='{"fetch":{"transport":"http","url":"http://localhost:8080/mcp"}}'
```

## Referencia de keys (selección)

- `llm.provider`, `llm.model`, `llm.base_url`, `llm.api_key`
- `agent.disable_action_fallback`, `agent.warn_on_action_fallback`
- `memory.enabled`, `memory.provider`, `memory.qdrant_addr`
- `mcp.servers.<name>.transport`, `mcp.servers.<name>.url`
- `agents.<agent_id>.agent_card_url`, `agents.<agent_id>.grpc_addr`, `agents.<agent_id>.http_url`, `agents.<agent_id>.labels`
- `telemetry.exporter`, `telemetry.otlp_endpoint`, `telemetry.otlp_insecure`
- `runtime.approval_sweep_interval_seconds`, `runtime.approval_sweep_timeout_seconds`
- `governance.approval_timeout_seconds`
- `discovery.order` (opcional; por defecto: `config, well_known, registry`)
- `discovery.registry_url` (opcional; habilita RegistryProvider)
- `discovery.registry_token` (opcional; token bearer)
- `discovery.auto_register` (opcional; registra el agente en registry si está habilitado)
- `discovery.heartbeat_seconds` (opcional; intervalo para auto-register, por defecto interno 10s)
