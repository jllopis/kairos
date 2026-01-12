# Configuration Guide

Kairos configuration can be provided via file, environment variables, and CLI
flags. The final configuration follows this precedence order:

1) Defaults
2) File (`~/.kairos/settings.json`, `./.kairos/settings.json`, or `XDG_CONFIG_HOME/kairos/settings.json`)
3) Environment variables (`KAIROS_*`)
4) CLI overrides (`--config`, `--set`)

## File configuration

Create a `settings.json` file in one of the supported paths:

- `./.kairos/settings.json`
- `$HOME/.kairos/settings.json`
- `$XDG_CONFIG_HOME/kairos/settings.json`

Example:

```json
{
  "llm": {
    "provider": "ollama",
    "model": "qwen2.5-coder:7b-instruct-q5_K_M"
  },
  "telemetry": {
    "exporter": "stdout"
  },
  "runtime": {
    "approval_sweep_interval_seconds": 30,
    "approval_sweep_timeout_seconds": 5
  },
  "governance": {
    "approval_timeout_seconds": 300
  },
  "mcp": {
    "servers": {
      "fetch": {
        "transport": "stdio",
        "command": "docker",
        "args": ["run", "-i", "--rm", "mcp/fetch"]
      }
    }
  }
}
```

## Environment variables

Environment variables map to config keys by:

1) Prefix: `KAIROS_`
2) Lowercase
3) `_` becomes `.`

Examples:

```bash
KAIROS_LLM_PROVIDER=ollama
KAIROS_LLM_MODEL=qwen2.5-coder:7b-instruct-q5_K_M
KAIROS_TELEMETRY_EXPORTER=stdout
KAIROS_RUNTIME_APPROVAL_SWEEP_INTERVAL_SECONDS=30
KAIROS_GOVERNANCE_APPROVAL_TIMEOUT_SECONDS=300
```

## CLI overrides

CLI supports a config path and repeatable key overrides:

- `--config=/path/to/settings.json`
- `--set key=value`

Examples:

```bash
go run ./examples/basic-agent --config=./.kairos/settings.json \
  --set llm.provider=ollama \
  --set telemetry.exporter=stdout
```

JSON values can be passed with `--set`:

```bash
go run ./examples/mcp-agent \
  --set mcp.servers='{"fetch":{"transport":"http","url":"http://localhost:8080/mcp"}}'
```

## Key reference (selected)

- `llm.provider`, `llm.model`, `llm.base_url`, `llm.api_key`
- `agent.disable_action_fallback`, `agent.warn_on_action_fallback`
- `memory.enabled`, `memory.provider`, `memory.qdrant_addr`
- `mcp.servers.<name>.transport`, `mcp.servers.<name>.url`
- `telemetry.exporter`, `telemetry.otlp_endpoint`, `telemetry.otlp_insecure`
- `runtime.approval_sweep_interval_seconds`, `runtime.approval_sweep_timeout_seconds`
- `governance.approval_timeout_seconds`
