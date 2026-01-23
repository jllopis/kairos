# Playbook 07 - MCP

Goal: discover and call MCP tools from the agent.

Incremental reuse:

- Add `internal/mcp` for MCP client setup and tool discovery.

What to implement:

- Run the MCP server from `examples/mcp-http-server` in a separate terminal.
- Configure MCP servers via config (`cfg.MCP.Servers`).
- Load MCP clients with `agent.WithMCPServerConfigs`.
- List tools with `ag.MCPTools(ctx)`.
- Call a tool either via the agent or via a direct MCP client.
- Reuse provider/config wiring from step 02 via shared helpers.

Suggested checks:

- MCP tools are discovered.
- A tool call succeeds and returns output.

Manual tests:

- "List the MCP tools you can use."

Expected behavior:

- The agent reports discovered MCP tools.
- A direct MCP call (e.g., echo) returns a response.

Checklist:

- [ ] MCP client connects via config.
- [ ] Tools are listed and callable.

References:

- `examples/05-mcp-agent`
- `examples/07-multi-agent-mcp`
- `examples/mcp-http-server`
- `pkg/mcp`
