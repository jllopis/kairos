package mcp

import (
	"context"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client wraps the mcp-go client to provide Kairos-specific functionality.
type Client struct {
	mcpClient client.MCPClient
	// We might want to store tools cache here later
}

// NewClient creates a new Client with the given MCP client implementation.
func NewClient(c client.MCPClient) *Client {
	return &Client{
		mcpClient: c,
	}
}

// NewClientWithStdio creates a new MCP client that connects via Stdio.
func NewClientWithStdio(command string, args []string) (*Client, error) {
	return NewClientWithStdioProtocol(command, args, mcp.LATEST_PROTOCOL_VERSION)
}

// NewClientWithStdioProtocol creates a new MCP client that connects via Stdio using a specified protocol version.
func NewClientWithStdioProtocol(command string, args []string, protocolVersion string) (*Client, error) {
	if protocolVersion == "" {
		protocolVersion = mcp.LATEST_PROTOCOL_VERSION
	}
	// client.NewStdioMCPClient returns a *client.Client which implements mcp.Client
	stdioClient, err := client.NewStdioMCPClient(command, nil, args...)
	if err != nil {
		return nil, err
	}

	// Start the client (which starts the subprocess)
	if err := stdioClient.Start(context.Background()); err != nil {
		return nil, err
	}

	// Initialize the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = protocolVersion
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "kairos-client",
		Version: "0.1.0",
	}

	_, err = stdioClient.Initialize(ctx, initRequest)
	if err != nil {
		return nil, err
	}

	return &Client{
		mcpClient: stdioClient,
	}, nil
}

// ListTools retrieves the list of tools available on the server.
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	req := mcp.ListToolsRequest{}
	resp, err := c.mcpClient.ListTools(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.Tools, nil
}

// CallTool executes a tool on the server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	return c.mcpClient.CallTool(ctx, req)
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.mcpClient.Close()
}
