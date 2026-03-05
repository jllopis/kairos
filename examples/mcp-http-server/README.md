<!--
Copyright 2026 © The Kairos Authors
SPDX-License-Identifier: Apache-2.0
-->

# MCP HTTP Server Example

Standalone MCP server using the Kairos streamable HTTP transport.

## What it demonstrates

- Creating an MCP server with `mcp.NewServer`
- Registering a simple `echo` tool
- Serving over streamable HTTP (SSE-based)

## Running

```bash
# Optional: set listen address (default: localhost:8080)
export KAIROS_MCP_HTTP_ADDR="localhost:8080"

go run .
```

The server exposes an MCP-compatible streamable HTTP endpoint. Connect any MCP client to `http://localhost:8080` to invoke the `echo` tool.

## Related

- [MCP Protocol docs](../../docs/protocols/MCP.md)
- [05-mcp-agent](../05-mcp-agent/) — Agent that connects to an MCP server
- [07-multi-agent-mcp](../07-multi-agent-mcp/) — Multiple agents sharing MCP tools
