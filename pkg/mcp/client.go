package mcp

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	defaultTimeout  = 10 * time.Second
	defaultRetries  = 2
	defaultBackoff  = 200 * time.Millisecond
	defaultCacheTTL = 30 * time.Second
)

// ClientOption customizes the MCP client wrapper behavior.
type ClientOption func(*Client)

// WithTimeout sets the per-request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithRetry configures retry count and backoff.
func WithRetry(retries int, backoff time.Duration) ClientOption {
	return func(c *Client) {
		if retries >= 0 {
			c.maxRetries = retries
		}
		if backoff > 0 {
			c.backoff = backoff
		}
	}
}

// WithToolCacheTTL sets the tool discovery cache TTL. Use 0 to disable caching.
func WithToolCacheTTL(ttl time.Duration) ClientOption {
	return func(c *Client) {
		if ttl >= 0 {
			c.cacheTTL = ttl
		}
	}
}

// Client wraps the mcp-go client to provide Kairos-specific functionality.
type Client struct {
	mcpClient  client.MCPClient
	timeout    time.Duration
	maxRetries int
	backoff    time.Duration
	cacheTTL   time.Duration

	mu          sync.Mutex
	toolsCache  []mcp.Tool
	cacheExpiry time.Time
}

// NewClient creates a new Client with the given MCP client implementation.
func NewClient(c client.MCPClient, opts ...ClientOption) *Client {
	client := &Client{
		mcpClient:  c,
		timeout:    defaultTimeout,
		maxRetries: defaultRetries,
		backoff:    defaultBackoff,
		cacheTTL:   defaultCacheTTL,
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

// NewClientWithStdio creates a new MCP client that connects via Stdio.
func NewClientWithStdio(command string, args []string, opts ...ClientOption) (*Client, error) {
	return NewClientWithStdioProtocol(command, args, mcp.LATEST_PROTOCOL_VERSION, opts...)
}

// NewClientWithStdioProtocol creates a new MCP client that connects via Stdio using a specified protocol version.
func NewClientWithStdioProtocol(command string, args []string, protocolVersion string, opts ...ClientOption) (*Client, error) {
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

	return NewClient(stdioClient, opts...), nil
}

// ListTools retrieves the list of tools available on the server.
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	if cached := c.cachedTools(); cached != nil {
		return cached, nil
	}
	req := mcp.ListToolsRequest{}
	resp, err := c.listToolsWithRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	c.storeTools(resp.Tools)
	return resp.Tools, nil
}

// CallTool executes a tool on the server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]interface{}) (*mcp.CallToolResult, error) {
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	return c.callToolWithRetry(ctx, req)
}

// Close closes the client connection.
func (c *Client) Close() error {
	return c.mcpClient.Close()
}

func (c *Client) cachedTools() []mcp.Tool {
	if c.cacheTTL == 0 {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.toolsCache) == 0 || time.Now().After(c.cacheExpiry) {
		return nil
	}
	out := make([]mcp.Tool, len(c.toolsCache))
	copy(out, c.toolsCache)
	return out
}

func (c *Client) storeTools(tools []mcp.Tool) {
	if c.cacheTTL == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.toolsCache = make([]mcp.Tool, len(tools))
	copy(c.toolsCache, tools)
	c.cacheExpiry = time.Now().Add(c.cacheTTL)
}

func (c *Client) listToolsWithRetry(ctx context.Context, req mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	var lastErr error
	attempts := c.maxRetries + 1
	for i := 0; i < attempts; i++ {
		reqCtx, cancel := c.withTimeout(ctx)
		res, err := c.mcpClient.ListTools(reqCtx, req)
		cancel()
		if err == nil {
			return res, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		lastErr = err
		if i == attempts-1 {
			break
		}
		if err := c.sleepBackoff(ctx, i); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) callToolWithRetry(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var lastErr error
	attempts := c.maxRetries + 1
	for i := 0; i < attempts; i++ {
		reqCtx, cancel := c.withTimeout(ctx)
		res, err := c.mcpClient.CallTool(reqCtx, req)
		cancel()
		if err == nil {
			return res, nil
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		lastErr = err
		if i == attempts-1 {
			break
		}
		if err := c.sleepBackoff(ctx, i); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) sleepBackoff(ctx context.Context, attempt int) error {
	wait := c.backoff * time.Duration(1<<attempt)
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
