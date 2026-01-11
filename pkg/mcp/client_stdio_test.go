package mcp

import (
	"context"
	"os"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const mcpStdioHelperEnv = "KAIROS_MCP_STDIO_HELPER"

func TestHelperMCPStdioServer(t *testing.T) {
	if os.Getenv(mcpStdioHelperEnv) != "1" {
		return
	}

	server := mcpserver.NewMCPServer("test-stdio", "1.0.0")
	server.AddTool(mcpgo.NewTool("ping"), func(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: "ok"}},
		}, nil
	})

	if err := mcpserver.ServeStdio(server); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestClient_Stdio_ListToolsAndCall(t *testing.T) {
	t.Setenv(mcpStdioHelperEnv, "1")

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}

	client, err := NewClientWithStdioProtocol(exe, []string{"-test.run", "TestHelperMCPStdioServer"}, mcpgo.LATEST_PROTOCOL_VERSION)
	if err != nil {
		t.Fatalf("NewClientWithStdioProtocol error: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) == 0 || tools[0].Name != "ping" {
		t.Fatalf("Expected tool 'ping', got %+v", tools)
	}

	result, err := client.CallTool(context.Background(), "ping", map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("Expected successful tool result, got %+v", result)
	}
}
