// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package pool provides a shared MCP connection pool for multi-agent scenarios.
//
// In enterprise deployments where multiple agents need access to the same MCP servers,
// creating individual connections per agent is inefficient. This package provides:
//
//   - Connection pooling: Reuse MCP client connections across agents
//   - Lifecycle management: Centralized start/stop/health for MCP servers
//   - Reference counting: Automatic cleanup when no agents use a connection
//   - Observability: Unified metrics for all MCP connections
//
// Example usage:
//
//	pool := pool.New(
//	    pool.WithMaxConnectionsPerServer(5),
//	    pool.WithHealthCheckInterval(30 * time.Second),
//	)
//
//	// Register MCP servers
//	pool.RegisterStdio("filesystem", "npx", []string{"-y", "@anthropic/mcp-server-filesystem"})
//	pool.RegisterHTTP("github", "http://localhost:8080/mcp")
//
//	// Get connections for agents
//	client1, _ := pool.Get(ctx, "filesystem")
//	client2, _ := pool.Get(ctx, "filesystem") // Reuses connection
//
//	// Release when done
//	pool.Release("filesystem", client1)
//	pool.Release("filesystem", client2)
package pool

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jllopis/kairos/pkg/mcp"
)

var (
	// ErrPoolClosed is returned when operations are attempted on a closed pool.
	ErrPoolClosed = errors.New("mcp pool is closed")

	// ErrServerNotFound is returned when requesting a connection to an unregistered server.
	ErrServerNotFound = errors.New("mcp server not found in pool")

	// ErrMaxConnectionsReached is returned when the pool cannot create more connections.
	ErrMaxConnectionsReached = errors.New("maximum connections reached for server")

	// ErrInvalidServerConfig is returned when server configuration is invalid.
	ErrInvalidServerConfig = errors.New("invalid server configuration")
)

// ServerType indicates how to connect to an MCP server.
type ServerType int

const (
	// ServerTypeStdio connects via stdio (subprocess).
	ServerTypeStdio ServerType = iota
	// ServerTypeHTTP connects via Streamable HTTP.
	ServerTypeHTTP
)

// ServerConfig holds the configuration for an MCP server.
type ServerConfig struct {
	// Name is the logical identifier for this server.
	Name string

	// Type indicates the connection method.
	Type ServerType

	// For stdio servers
	Command string
	Args    []string

	// For HTTP servers
	URL string

	// Env holds environment variables for stdio servers.
	Env map[string]string

	// MaxConnections limits concurrent connections (0 = unlimited).
	MaxConnections int

	// ClientOptions are applied to each client created for this server.
	ClientOptions []mcp.ClientOption
}

// pooledClient wraps an MCP client with reference counting.
type pooledClient struct {
	client   *mcp.Client
	refCount int32
	server   string
	created  time.Time
}

// Pool manages shared MCP connections across multiple agents.
type Pool struct {
	mu      sync.RWMutex
	servers map[string]*ServerConfig
	clients map[string][]*pooledClient
	closed  atomic.Bool

	// Configuration
	maxPerServer        int
	healthCheckInterval time.Duration
	idleTimeout         time.Duration

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	totalConnections   atomic.Int64
	activeConnections  atomic.Int64
	connectionErrors   atomic.Int64
	healthChecksPassed atomic.Int64
	healthChecksFailed atomic.Int64
}

// PoolOption configures the connection pool.
type PoolOption func(*Pool)

// WithMaxConnectionsPerServer sets the default maximum connections per server.
func WithMaxConnectionsPerServer(max int) PoolOption {
	return func(p *Pool) {
		if max > 0 {
			p.maxPerServer = max
		}
	}
}

// WithHealthCheckInterval sets how often to check connection health.
func WithHealthCheckInterval(interval time.Duration) PoolOption {
	return func(p *Pool) {
		if interval > 0 {
			p.healthCheckInterval = interval
		}
	}
}

// WithIdleTimeout sets how long idle connections are kept before cleanup.
func WithIdleTimeout(timeout time.Duration) PoolOption {
	return func(p *Pool) {
		if timeout > 0 {
			p.idleTimeout = timeout
		}
	}
}

// New creates a new MCP connection pool.
func New(opts ...PoolOption) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Pool{
		servers:             make(map[string]*ServerConfig),
		clients:             make(map[string][]*pooledClient),
		maxPerServer:        10,
		healthCheckInterval: 30 * time.Second,
		idleTimeout:         5 * time.Minute,
		ctx:                 ctx,
		cancel:              cancel,
	}

	for _, opt := range opts {
		opt(p)
	}

	// Start background health checker
	p.wg.Add(1)
	go p.healthChecker()

	return p
}

// RegisterStdio registers an MCP server that connects via stdio.
func (p *Pool) RegisterStdio(name, command string, args []string, opts ...mcp.ClientOption) error {
	if name == "" || command == "" {
		return ErrInvalidServerConfig
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.Load() {
		return ErrPoolClosed
	}

	p.servers[name] = &ServerConfig{
		Name:          name,
		Type:          ServerTypeStdio,
		Command:       command,
		Args:          args,
		ClientOptions: opts,
	}

	return nil
}

// RegisterHTTP registers an MCP server that connects via Streamable HTTP.
func (p *Pool) RegisterHTTP(name, url string, opts ...mcp.ClientOption) error {
	if name == "" || url == "" {
		return ErrInvalidServerConfig
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.Load() {
		return ErrPoolClosed
	}

	p.servers[name] = &ServerConfig{
		Name:          name,
		Type:          ServerTypeHTTP,
		URL:           url,
		ClientOptions: opts,
	}

	return nil
}

// Register registers an MCP server with full configuration.
func (p *Pool) Register(config ServerConfig) error {
	if config.Name == "" {
		return ErrInvalidServerConfig
	}
	if config.Type == ServerTypeStdio && config.Command == "" {
		return ErrInvalidServerConfig
	}
	if config.Type == ServerTypeHTTP && config.URL == "" {
		return ErrInvalidServerConfig
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.Load() {
		return ErrPoolClosed
	}

	p.servers[config.Name] = &config
	return nil
}

// Unregister removes a server from the pool and closes all its connections.
func (p *Pool) Unregister(name string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed.Load() {
		return ErrPoolClosed
	}

	delete(p.servers, name)

	// Close all connections for this server
	if clients, ok := p.clients[name]; ok {
		for _, pc := range clients {
			_ = pc.client.Close()
			p.activeConnections.Add(-1)
		}
		delete(p.clients, name)
	}

	return nil
}

// Get retrieves a client connection for the specified server.
// If no idle connection is available, a new one is created.
func (p *Pool) Get(ctx context.Context, serverName string) (*mcp.Client, error) {
	if p.closed.Load() {
		return nil, ErrPoolClosed
	}

	p.mu.Lock()
	config, ok := p.servers[serverName]
	if !ok {
		p.mu.Unlock()
		return nil, fmt.Errorf("%w: %s", ErrServerNotFound, serverName)
	}

	// Try to find an existing connection with capacity
	clients := p.clients[serverName]
	for _, pc := range clients {
		// Simple approach: always increment ref count for existing client
		atomic.AddInt32(&pc.refCount, 1)
		p.mu.Unlock()
		return pc.client, nil
	}

	// Check if we can create a new connection
	maxConns := config.MaxConnections
	if maxConns == 0 {
		maxConns = p.maxPerServer
	}
	if len(clients) >= maxConns {
		p.mu.Unlock()
		return nil, ErrMaxConnectionsReached
	}
	p.mu.Unlock()

	// Create new connection outside the lock
	client, err := p.createClient(ctx, config)
	if err != nil {
		p.connectionErrors.Add(1)
		return nil, err
	}

	pc := &pooledClient{
		client:   client,
		refCount: 1,
		server:   serverName,
		created:  time.Now(),
	}

	p.mu.Lock()
	p.clients[serverName] = append(p.clients[serverName], pc)
	p.mu.Unlock()

	p.totalConnections.Add(1)
	p.activeConnections.Add(1)

	return client, nil
}

// Release decrements the reference count for a connection.
// The connection is not immediately closed but may be reused.
func (p *Pool) Release(serverName string, client *mcp.Client) {
	p.mu.RLock()
	clients := p.clients[serverName]
	p.mu.RUnlock()

	for _, pc := range clients {
		if pc.client == client {
			atomic.AddInt32(&pc.refCount, -1)
			return
		}
	}
}

// Close shuts down the pool and all connections.
func (p *Pool) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return ErrPoolClosed
	}

	p.cancel()
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	var errs []error
	for name, clients := range p.clients {
		for _, pc := range clients {
			if err := pc.client.Close(); err != nil {
				errs = append(errs, fmt.Errorf("closing %s: %w", name, err))
			}
		}
	}

	p.clients = nil
	p.servers = nil

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Stats returns current pool statistics.
func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	serverCount := len(p.servers)
	clientCount := 0
	for _, clients := range p.clients {
		clientCount += len(clients)
	}
	p.mu.RUnlock()

	return PoolStats{
		RegisteredServers:  serverCount,
		ActiveConnections:  int(p.activeConnections.Load()),
		TotalConnections:   int(p.totalConnections.Load()),
		ConnectionErrors:   int(p.connectionErrors.Load()),
		HealthChecksPassed: int(p.healthChecksPassed.Load()),
		HealthChecksFailed: int(p.healthChecksFailed.Load()),
	}
}

// PoolStats contains pool metrics.
type PoolStats struct {
	RegisteredServers  int
	ActiveConnections  int
	TotalConnections   int
	ConnectionErrors   int
	HealthChecksPassed int
	HealthChecksFailed int
}

// ListServers returns the names of all registered servers.
func (p *Pool) ListServers() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	names := make([]string, 0, len(p.servers))
	for name := range p.servers {
		names = append(names, name)
	}
	return names
}

// ServerInfo returns information about a registered server.
func (p *Pool) ServerInfo(name string) (ServerConfig, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	config, ok := p.servers[name]
	if !ok {
		return ServerConfig{}, false
	}
	return *config, true
}

func (p *Pool) createClient(ctx context.Context, config *ServerConfig) (*mcp.Client, error) {
	switch config.Type {
	case ServerTypeStdio:
		return mcp.NewClientWithStdio(config.Command, config.Args, config.Env, config.ClientOptions...)
	case ServerTypeHTTP:
		return mcp.NewClientWithStreamableHTTP(config.URL, config.ClientOptions...)
	default:
		return nil, fmt.Errorf("unknown server type: %d", config.Type)
	}
}

func (p *Pool) healthChecker() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.runHealthChecks()
		}
	}
}

func (p *Pool) runHealthChecks() {
	p.mu.RLock()
	toCheck := make([]*pooledClient, 0)
	for _, clients := range p.clients {
		toCheck = append(toCheck, clients...)
	}
	p.mu.RUnlock()

	for _, pc := range toCheck {
		// Simple health check: try to list tools
		ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
		_, err := pc.client.ListTools(ctx)
		cancel()

		if err != nil {
			p.healthChecksFailed.Add(1)
			// If no refs and failed, mark for cleanup
			if atomic.LoadInt32(&pc.refCount) == 0 {
				p.removeClient(pc)
			}
		} else {
			p.healthChecksPassed.Add(1)
		}
	}

	// Clean up idle connections
	p.cleanupIdle()
}

func (p *Pool) removeClient(pc *pooledClient) {
	p.mu.Lock()
	defer p.mu.Unlock()

	clients := p.clients[pc.server]
	for i, c := range clients {
		if c == pc {
			_ = c.client.Close()
			p.clients[pc.server] = append(clients[:i], clients[i+1:]...)
			p.activeConnections.Add(-1)
			return
		}
	}
}

func (p *Pool) cleanupIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for server, clients := range p.clients {
		remaining := clients[:0]
		for _, pc := range clients {
			// Keep if: has refs, or not idle long enough, or is the only connection
			isIdle := atomic.LoadInt32(&pc.refCount) == 0 && now.Sub(pc.created) > p.idleTimeout
			if !isIdle || len(clients) == 1 {
				remaining = append(remaining, pc)
			} else {
				_ = pc.client.Close()
				p.activeConnections.Add(-1)
			}
		}
		p.clients[server] = remaining
	}
}
