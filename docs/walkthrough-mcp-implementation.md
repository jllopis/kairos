# MCP Implementation Walkthrough

I have implemented the **Model Context Protocol (MCP)** using the `github.com/mark3labs/mcp-go` library, enabling Kairos agents to interact with external tools in a standardized way.

## Components Implemented

### 1. MCP Wrappers (`pkg/mcp`)

We created lightweight wrappers around `mcp-go` to integrate it into the Kairos ecosystem:

- **Client**: `pkg/mcp/client.go` provides a simplified `Client` struct and helpers to connect to MCP servers via Stdio or Streamable HTTP, including protocol selection when needed.
- **Server**: `pkg/mcp/server.go` allows creating local MCP servers and registering Go functions as tools, with Stdio or Streamable HTTP serving options.

### 2. Agent Integration (`pkg/agent`)

The `Agent` struct now supports an optional list of MCP clients.

- **Tools Discovery**: On every run, the agent queries connected MCP clients for available tools and filters them by `skills` (if configured).
- **Reasoning Loop**: The `Run` method has been upgraded to a loop (default max 10 turns) that:
    1. Sends user input + available tools to the LLM.
    2. If the LLM requests a tool call, the agent executes it via the appropriate MCP client.
    3. The tool result is fed back to the LLM to generate the final response.

### 2.1 Tool Argument Normalization

When a tool requires a `url` field and the LLM returns the action input as a plain URL
(instead of JSON), the MCP adapter maps that string to `{"url": "<value>"}` to satisfy
the schema.

### 3. LLM Provider Support (`pkg/llm`)

Updated `OllamaProvider` and basic interfaces to support:

- `Tools` definition in ChatRequest.
- `ToolCalls` in ChatResponse.

## Usage Example

### connecting an Agent to an MCP Server

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/core"
    "github.com/jllopis/kairos/pkg/llm"
    "github.com/jllopis/kairos/pkg/mcp"
)

func main() {
    // 1. Create a Stdio MCP Client (e.g., using the mcp-server-starter dist/index.js)
    serverPath := os.Getenv("MCP_SERVER_PATH")
    if serverPath == "" {
        log.Fatal("MCP_SERVER_PATH is required")
    }
    client, err := mcp.NewClientWithStdioProtocol("node", []string{serverPath}, "2024-11-05")
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 2. Initialize Agent with the Client
    llmProvider := llm.NewOllama("http://localhost:11434")
    a, err := agent.New("my-agent", llmProvider,
        agent.WithSkills([]core.Skill{{Name: "echo"}}),
        agent.WithMCPClients(client),
    )
    if err != nil {
        panic(err)
    }

    // 3. Run Agent
    response, err := a.Run(context.Background(), "Use the tools to solve this...")
    fmt.Println(response)
}
```

### runnable example (config-driven)

See `examples/mcp-agent/main.go` for a working end-to-end sample. It reads
`mcp.servers` from a `kairos.json`-style config file.

Example config (stdio):

```json
{
  "mcpServers": {
    "fetch": {
      "transport": "stdio",
      "command": "docker",
      "args": ["run", "-i", "--rm", "mcp/fetch"]
    }
  }
}
```

Example config (streamable HTTP):

```json
{
  "mcpServers": {
    "fetch-http": {
      "transport": "http",
      "url": "http://localhost:8080/mcp"
    }
  }
}
```

Place it in one of:
- `./.kairos/settings.json`
- `$HOME/.kairos/settings.json`
- `$XDG_CONFIG_HOME/kairos/settings.json`

## Verification

A new test `pkg/mcp/tool_adapter_test.go` verifies MCP tool mapping behavior.
Run it with:

```bash
go test -v ./pkg/mcp -run TestToolAdapter
```
