package demo

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	if err := waitForMCPServer(baseURL, 2*time.Second); err != nil {
		return nil, err
	}
	return mcp.NewClientWithStreamableHTTP(baseURL)
}

func waitForMCPServer(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("mcp server not ready at %s", baseURL)
}
