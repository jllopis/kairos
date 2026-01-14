# Governance: enforcement server-side y HITL

Este documento resume cómo se aplica la gobernanza en el servidor y cómo
funciona el flujo de aprobación humana (HITL).

## Objetivos

- Aplicar decisiones de `governance.PolicyEngine` en los handlers A2A.
- Permitir aprobaciones humanas cuando una acción es sensible.
- Mantener trazas y auditoría end-to-end.

## Fuera de alcance (por ahora)

- Autenticación/autorización completa (OIDC/mTLS).
- Flujos UI/CLI para aprobaciones (se rastrean en la fase UI/CLI).

## Enforcement en A2A

La evaluación ocurre en `SimpleHandler.SendMessage` y
`SimpleHandler.SendStreamingMessage`. El handler construye un
`governance.Action` con `Type: ActionAgent` y un `Name` estable, evalúa la
política y, si se deniega, devuelve `TASK_STATE_REJECTED` con la razón.

Se acepta metadata de identidad en la request: `caller`, `agent`, `tenant`.
El mismo flujo aplica a HTTP+JSON y JSON-RPC.

## Flujo HITL

Cuando una regla devuelve `effect: "pending"`, se inicia una aprobación.

1. El handler crea un `ApprovalRecord` en el `ApprovalStore`.
2. Se devuelve `TASK_STATE_INPUT_REQUIRED` con `approval_id` y
   `approval_expires_at` si hay timeout.
3. Un operador aprueba o rechaza la petición.
4. Las aprobaciones expiradas se rechazan por el sweeper o al acceder.

El `ApprovalStore` tiene backends in-memory y SQLite. Los timeouts se controlan
con `governance.approval_timeout_seconds` y los sweeps con
`runtime.approval_sweep_interval_seconds` y
`runtime.approval_sweep_timeout_seconds`.

Las aprobaciones quedan auditadas y vinculadas a la traza original.
