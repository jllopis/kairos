# 2. Tool System Implementation using Model Context Protocol (MCP)

Date: 2026-01-10

## Status

Accepted

## Context

The Kairos agent framework requires a standardized, extensible way for agents to discover and interact with external tools. Without a standard protocol:

- Every new tool requires custom integration code in the agent.
- Tools cannot be easily shared or reused across different agent implementations.
- There is no unified way to handle authentication, error reporting, and schema definition for tools.
- Users have to implement the glue code between their functions and the LLM's tool calling API manually every time.

We evaluated the following options:

1. **Custom "Native" Go Interface**: Define a `Tool` interface in Go (e.g., `Call(ctx, input) (output, error)`).
2. **OpenAPI / Swagger**: Use OpenAPI specs to define tools and HTTP for transport.
3. **Model Context Protocol (MCP)**: An open protocol that standardizes how AI models interact with external data and tools.

## Decision

We decided to implement the **Model Context Protocol (MCP)** using the `github.com/mark3labs/mcp-go` library.

### Justification

1. **Standardization**: MCP provides a robust, standardized protocol for tool discovery (`ListTools`), execution (`CallTool`), and resource management. This avoids "reinventing the wheel" for common problems like JSON-RPC message handling and error management.
2. **Ecosystem Compatibility**: By adopting MCP, Kairos agents can potentially interact with *any* MCP-compliant server (e.g., verified servers for database access, file system, APIs), not just tools written specifically for Kairos.
3. **Future-Proofing**: MCP supports more than just tools; it handles "Resources" (context data) and "Prompts", aligned with our roadmap for a more context-aware agent.
4. **Library Selection (`mcp-go`)**:
    - `mark3labs/mcp-go` is a mature, community-driven implementation of the MCP spec in Go.
    - It handles the low-level JSON-RPC 2.0 transport (Stdio, SSE), allowing us to focus on the agent logic.
    - It provides clean interfaces for both Client (consuming tools) and Server (exposing tools).

## Consequences

### Positive

- **Extensibility**: Users can add tools by simply pointing the agent to an MCP server executable (e.g., a Python script or a Node.js app), without recompiling the agent.
- **Simplification**: The `Agent` loop does not need to know the implementation details of a tool; it just deals with the standardized MCP interface.
- **Interoperability**: Kairos agents become compatible with the broader MCP ecosystem.

### Negative

- **Complexity**: Introducing a full protocol adds slight overhead compared to a direct Go function call.
- **Dependency**: We take a dependency on `github.com/mark3labs/mcp-go`.
- **Runtime Overheads**: Running MCP servers as subprocesses (Stdio transport) introduces process management complexity and serialization/deserialization overhead.

## Implementation Details

The implementation involves:

1. **Wrappers**: Lightweight wrappers in `pkg/mcp` to adapt `mcp-go` types to our domain.
2. **Agent Integration**: The `Agent` struct holds references to `mcp.Client` instances.
3. **Discovery Loop**: On `Run`, the agent queries connected clients for `ListTools` and converts them into the LLM's native tool definition format.
4. **Execution Loop**: The agent's reasoning loop handles LLM requests for tool calls by routing them to the appropriate MCP client.
