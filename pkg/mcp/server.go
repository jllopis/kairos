package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps the mcp-go server to provide Kairos-specific functionality.
type Server struct {
	mcpServer *server.MCPServer
}

// NewServer creates a new MCP server.
func NewServer(name, version string) *Server {
	return &Server{
		mcpServer: server.NewMCPServer(name, version),
	}
}

// RegisterTool registers a tool with the server.
func (s *Server) RegisterTool(name, description string, schema interface{}, handler func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error)) {
	tool := mcp.NewTool(name, mcp.WithDescription(description))

	s.mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, _ := request.Params.Arguments.(map[string]interface{})
		return handler(ctx, args)
	})
}

// ServeStdio starts the server on Stdio.
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpServer)
}
