# Walkthrough: Agent Discovery Patterns

Este walkthrough detalla la implementacion del discovery de agentes con tres patrones:
configurado, well-known y registry externo. El objetivo es mantener A2A-compliance
sin imponer discovery dinamico.

## Principios

- A2A no define discovery dinamico por razones de seguridad y neutralidad.
- Discovery es infraestructura; A2A comienza al invocar un agente.
- Debe ser reemplazable si aparece un estandar A2A en el futuro.

## Patrones

1) Discovery por configuracion (determinista)
   - Fuente: config local.
   - Default en enterprise.
   - Facil de securizar.

2) Discovery well-known (semi-dinamico)
   - Fuente: URL base -> `/.well-known/agent-card.json`.
   - Auto-descriptivo y facil de documentar.

3) Registry externo (no estandar)
   - Fuente: registry propio / k8s / Consul / DB.
   - A2A empieza despues (no es parte del protocolo).

## Fase 1: Core discovery + config

- Crear `pkg/discovery` con:
  - `Provider` y `Resolver`.
  - `AgentEndpoint` con campos minimos.
- Configurar orden:
  - `discovery.order` opcional.
  - Default: `config -> well_known -> registry`.
- Tests basicos del resolver.

## Fase 2: Providers iniciales

- `ConfigProvider` (config local).
- `WellKnownProvider` (AgentCard fetch).
- Dedupe por `agent_card_url` (o `name+url`).

## Fase 3: Registry opcional

- `RegistryProvider` opt-in (via `discovery.registry_url`).
- TTL + expiracion con helper `RegistryServer`.
- Auth opcional con `discovery.registry_token`.
- Auto-register opcional desde agentes (`discovery.auto_register`).
- Heartbeat configurable (`discovery.heartbeat_seconds`).

### Ejemplo rapido (CLI)

Arranca el registry local:

```
kairos registry serve --addr :9900 --ttl 30s
```

Registra un agente:

```
curl -s -X POST http://localhost:9900/v1/agents \\
  -H 'Content-Type: application/json' \\
  -d '{
    "name":"orchestrator",
    "agent_card_url":"http://127.0.0.1:9140/.well-known/agent-card.json",
    "grpc_addr":"127.0.0.1:9030",
    "http_url":"http://127.0.0.1:8080",
    "labels":{"env":"local","tier":"core"}
  }'
```

Lista agentes:

```
curl -s http://localhost:9900/v1/agents | jq
```

### Auto-register (opcional)

Cuando `discovery.auto_register=true`, el agente se registra en el registry
de forma periodica usando `discovery.heartbeat_seconds` (default interno 10s).

Ejemplo:

```json
{
  "discovery": {
    "registry_url": "http://localhost:9900",
    "auto_register": true,
    "heartbeat_seconds": 10
  }
}
```

## Fase 4: Wiring CLI/UI

- CLI `agents list` usa resolver.
- UI `Agents` usa resolver.

## Fase 5: Docs

- Actualizar `docs/CONFIGURATION.md` con `discovery.*`.
- Referenciar ADR 0004 en `docs/ARCHITECTURE.md`.
