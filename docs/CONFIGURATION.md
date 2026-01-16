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

Las variables de entorno permiten configurar Kairos sin archivos. El mapeo funciona así:

1. Prefijo obligatorio: `KAIROS_`
2. Después del prefijo, `_` se convierte en `.` para navegar la estructura
3. El nombre se convierte a minúsculas

Por ejemplo:
- `KAIROS_LLM_MODEL` → `llm.model`
- `KAIROS_LLM_BASE_URL` → `llm.base_url`
- `KAIROS_TELEMETRY_OTLP_ENDPOINT` → `telemetry.otlp_endpoint`

Ejemplos de uso:

```bash
# Configurar LLM
export KAIROS_LLM_PROVIDER=ollama
export KAIROS_LLM_MODEL=llama3.2
export KAIROS_LLM_BASE_URL=http://localhost:11434

# Configurar telemetría
export KAIROS_TELEMETRY_EXPORTER=otlp
export KAIROS_TELEMETRY_OTLP_ENDPOINT=localhost:4317

# Ejecutar con variables de entorno
KAIROS_LLM_MODEL=qwen2.5-coder:7b go run ./examples/02-basic-agent
```

Las variables de entorno tienen precedencia sobre el archivo de configuración, pero los flags CLI (`--set`) tienen precedencia sobre ambos.

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

## Config Layering (Perfiles de entorno)

Para proyectos enterprise, Kairos soporta **config layering** con perfiles de entorno. Esto permite mantener una configuración base y sobrescribirla según el entorno (dev, staging, prod).

### Estructura de archivos

```
config/
├── config.yaml           # Configuración base
├── config.dev.yaml       # Sobrescrituras para desarrollo
├── config.staging.yaml   # Sobrescrituras para staging
└── config.prod.yaml      # Sobrescrituras para producción
```

### Ejemplo de configuración

**config.yaml (base)**:
```yaml
llm:
  provider: "ollama"
  model: "llama3.1"
  base_url: "http://localhost:11434"

log:
  level: "info"

telemetry:
  exporter: "stdout"
```

**config.dev.yaml**:
```yaml
llm:
  provider: "mock"  # Sin LLM real en desarrollo

log:
  level: "debug"
```

**config.prod.yaml**:
```yaml
llm:
  provider: "openai"
  api_key: "${OPENAI_API_KEY}"  # Usar variable de entorno

log:
  level: "warn"

telemetry:
  exporter: "otlp"
  otlp_endpoint: "collector.internal:4317"
```

### Uso desde CLI

```bash
# Cargar config.yaml + config.dev.yaml
kairos run --config config/config.yaml --profile dev

# Equivalente con --env
kairos run --config config/config.yaml --env dev

# Producción
kairos run --config config/config.yaml --profile prod
```

### Uso programático

```go
import "github.com/jllopis/kairos/pkg/config"

// Cargar con perfil específico
cfg, err := config.LoadWithProfile("config/config.yaml", "dev")

// O desde argumentos CLI
cfg, err := config.LoadWithCLI(os.Args[1:])
```

### Orden de precedencia

Con layering, el orden completo es:

1. Valores por defecto del framework
2. Archivo base (`config.yaml`)
3. Archivo de perfil (`config.dev.yaml`, `config.prod.yaml`, etc.)
4. Variables de entorno (`KAIROS_*`)
5. Sobrescrituras CLI (`--set key=value`)

El merge es **profundo**: las claves del perfil sobrescriben las del base, pero las claves no especificadas se heredan.

### Buenas prácticas

| Archivo | Qué incluir |
|---------|-------------|
| `config.yaml` | Valores por defecto sensatos para todos los entornos |
| `config.dev.yaml` | Mock providers, debug logging, endpoints locales |
| `config.prod.yaml` | Providers reales, warn logging, endpoints de producción |

**Consejo**: No incluyas secrets en archivos. Usa variables de entorno o un gestor de secrets.

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
