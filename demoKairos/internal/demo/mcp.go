package demo

import (
	"context"
	"fmt"

	"github.com/jllopis/kairos/pkg/mcp"
	mcpgo "github.com/mark3labs/mcp-go/mcp"
)

// MCPServer wraps an MCP Streamable HTTP server.
type MCPServer struct {
	addr   string
	server *mcp.Server
}

// StartMCPServer starts a Streamable HTTP MCP server on the given address.
func StartMCPServer(name, version, addr string) (*MCPServer, error) {
	if addr == "" {
		addr = "127.0.0.1:9041"
	}
	srv := mcp.NewServer(name, version)
	httpSrv := srv.StreamableHTTPServer()
	go func() {
		_ = httpSrv.Start(addr)
	}()
	return &MCPServer{addr: addr, server: srv}, nil
}

// RegisterTool registers a tool on the MCP server.
func (s *MCPServer) RegisterTool(name, description string, handler func(ctx context.Context, args map[string]interface{}) (*mcpgo.CallToolResult, error)) {
	s.server.RegisterTool(name, description, nil, handler)
}

// BaseURL returns the HTTP URL for MCP client connections.
func (s *MCPServer) BaseURL() string {
	return fmt.Sprintf("http://%s/mcp", s.addr)
}

// NewMCPClient creates a client connected to the server base URL.
func NewMCPClient(baseURL string) (*mcp.Client, error) {
	return mcp.NewClientWithStreamableHTTP(baseURL)
}
