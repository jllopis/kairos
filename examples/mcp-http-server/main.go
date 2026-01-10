package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jllopis/kairos/pkg/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

func main() {
	addr := os.Getenv("KAIROS_MCP_HTTP_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	server := mcp.NewServer("kairos-http-tools", "0.1.0")
	server.RegisterTool("echo", "Echo back a message", nil, func(ctx context.Context, args map[string]interface{}) (*mcpgo.CallToolResult, error) {
		message := fmt.Sprint(args["message"])
		return &mcpgo.CallToolResult{
			Content: []mcpgo.Content{
				mcpgo.TextContent{Type: "text", Text: message},
			},
		}, nil
	})

	log.Printf("Starting MCP streamable HTTP server on %s", addr)
	if err := server.ServeStreamableHTTP(addr); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
