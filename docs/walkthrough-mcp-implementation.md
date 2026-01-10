# MCP Implementation Walkthrough

I have implemented the **Model Context Protocol (MCP)** using the `github.com/mark3labs/mcp-go` library, enabling Kairos agents to interact with external tools in a standardized way.

## Components Implemented

### 1. MCP Wrappers (`pkg/mcp`)

We created lightweight wrappers around `mcp-go` to integrate it into the Kairos ecosystem:

- **Client**: `pkg/mcp/client.go` provides a simplified `Client` struct and a helper `NewClientWithStdio` to connect to local MCP servers via Stdio.
- **Server**: `pkg/mcp/server.go` allows creating local MCP servers and registering Go functions as tools.

### 2. Agent Integration (`pkg/agent`)

The `Agent` struct now supports an optional list of MCP clients.

- **Tools Discovery**: On every run, the agent queries connected MCP clients for available tools.
- **Reasoning Loop**: The `Run` method has been upgraded to a loop (max 5 turns) that:
    1. Sends user input + available tools to the LLM.
    2. If the LLM requests a tool call, the agent executes it via the appropriate MCP client.
    3. The tool result is fed back to the LLM to generate the final response.

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
    "github.com/jllopis/kairos/pkg/agent"
    "github.com/jllopis/kairos/pkg/llm"
    "github.com/jllopis/kairos/pkg/mcp"
)

func main() {
    // 1. Create a Stdio MCP Client (e.g., connecting to a local python server)
    client, err := mcp.NewClientWithStdio("python", []string{"server.py"})
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // 2. Initialize Agent with the Client
    llmProvider := llm.NewOllama("http://localhost:11434")
    a, err := agent.New("my-agent", llmProvider, agent.WithMCPClient(client))
    if err != nil {
        panic(err)
    }

    // 3. Run Agent
    response, err := a.Run(context.Background(), "Use the tools to solve this...")
    fmt.Println(response)
}
```

## Verification

A new test `pkg/agent/agent_tool_test.go` was created to verify the entire flow using a Mock MCP Client and Mock LLM.
Run it with:

```bash
go test -v ./pkg/agent/ -run TestAgent_Run_ToolLoop
```
