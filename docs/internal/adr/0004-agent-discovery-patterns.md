# ADR 0004: Agent Discovery Patterns

## Estado

Propuesto.

## Contexto

Kairos necesita discovery de agentes sin imponer un unico mecanismo. A2A no
define discovery dinamico por razones de seguridad, neutralidad de
infraestructura y separacion de responsabilidades. Los entornos enterprise
prefieren discovery determinista y auditado, mientras que otros entornos
requieren mecanismos mas dinamicos.

## Decision

Adoptar tres patrones de discovery, cada uno como provider pluggable:

1) **Discovery por configuracion (determinista)**  
   - Fuente: config local (equivalente al modelo actual).
   - Seguro, auditado, enterprise-friendly.
   - Totalmente A2A-compliant.

2) **Discovery well-known (semi-dinamico)**  
   - Fuente: URL base -> `/.well-known/agent-card.json`.
   - Auto-descriptivo, facil de documentar.
   - Sigue siendo determinista en origen (se conoce la base URL).

3) **Registry externo (no estandar)**  
   - Fuente: registry propio / k8s / Consul / DB interna.
   - No forma parte de A2A; A2A empieza al invocar el agente.
   - Opt-in via `discovery.registry_url` (token opcional).
   - Auto-register opcional via `discovery.auto_register` + `discovery.heartbeat_seconds`.

### Resolver y orden

Se define un `Resolver` que combina providers en orden configurable:

- **Default**: `Config -> WellKnown -> Registry`.
- **Preferencia explicita**: se permite configurar el orden (por ejemplo
  `discovery.order=[config, well_known, registry]`).
- Si no hay preferencia, se aplica el orden por defecto.

## Consecuencias

- No se fuerza discovery dinamico: se respeta el modelo enterprise.
- Se habilita substitucion futura si A2A define un estandar de discovery.
- El discovery queda desacoplado del protocolo A2A.

## Alcance inicial (propuesto)

- `pkg/discovery` con interfaz `Provider` y tipo `AgentEndpoint`.
- Providers iniciales: `ConfigProvider`, `WellKnownProvider`, `RegistryProvider` (opt-in).
- CLI/UI consumen el `Resolver`, no variables sueltas.
