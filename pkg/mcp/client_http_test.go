package mcp

import (
	"context"
	"testing"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func TestClient_StreamableHTTP_ListTools(t *testing.T) {
	server := mcpserver.NewMCPServer("test-http", "1.0.0")
	server.AddTool(mcpgo.NewTool("ping"), func(ctx context.Context, _ mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{mcpgo.TextContent{Type: "text", Text: "ok"}},
		}, nil
	})

	httpServer := mcpserver.NewTestStreamableHTTPServer(server)
	defer httpServer.Close()

	client, err := NewClientWithStreamableHTTPProtocol(httpServer.URL, mcpgo.LATEST_PROTOCOL_VERSION)
	if err != nil {
		t.Fatalf("NewClientWithStreamableHTTPProtocol error: %v", err)
	}
	defer client.Close()

	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools error: %v", err)
	}
	if len(tools) == 0 || tools[0].Name != "ping" {
		t.Fatalf("Expected tool 'ping', got %+v", tools)
	}
}
