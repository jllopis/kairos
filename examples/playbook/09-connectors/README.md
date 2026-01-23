# Playbook 09 - Connectors

Goal: generate tools from OpenAPI, GraphQL, gRPC, and SQL specs.

Incremental reuse:

- Add `internal/connectors` for connector setup and tool wiring.

What to implement:

- OpenAPI:
  - Use `connectors.NewFromBytes` + a local HTTP server.
  - List tools with `Tools()` and call `Execute`.
- GraphQL:
  - Prefer `NewGraphQLConnectorFromSchema` for a local schema.
  - Or use `NewGraphQLConnector` against a real endpoint.
- gRPC:
  - Prefer `NewGRPCConnectorFromServices` for a local stub.
  - Or use `NewGRPCConnector` against a server with reflection.
- SQL:
  - Use SQLite in-memory and create a schema.
  - `NewSQLConnector(db, "sqlite", connectors.WithSQLReadOnly())`.
- Reuse provider/config wiring from step 02 via shared helpers.

Integration note:

- `connector.Tools()` returns `[]core.Tool` ready for `agent.WithTools`.
- Use `tool.ToolDefinition()` if you need the `llm.Tool` schema.

Suggested checks:

- Each connector lists tools and executes at least one call.

Manual tests:

- OpenAPI: call one GET and one POST operation.
- GraphQL: run one query and one mutation.
- gRPC: call one unary method.
- SQL: list records and fetch by primary key.

Expected behavior:

- Tools are generated and invocable for every connector.

Checklist:

- [ ] `Tools()` returns `core.Tool` for all connectors.
- [ ] `Execute` handles args and returns results.
- [ ] Tool schemas are accessible via `ToolDefinition()`.

References:

- `examples/17-openapi-connector`
- `examples/19-graphql-connector`
- `pkg/connectors`
