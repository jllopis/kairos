// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package pool

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewPool(t *testing.T) {
	p := New()
	defer p.Close()

	stats := p.Stats()
	if stats.RegisteredServers != 0 {
		t.Errorf("expected 0 servers, got %d", stats.RegisteredServers)
	}
	if stats.ActiveConnections != 0 {
		t.Errorf("expected 0 active connections, got %d", stats.ActiveConnections)
	}
}

func TestPoolOptions(t *testing.T) {
	p := New(
		WithMaxConnectionsPerServer(5),
		WithHealthCheckInterval(10*time.Second),
		WithIdleTimeout(1*time.Minute),
	)
	defer p.Close()

	if p.maxPerServer != 5 {
		t.Errorf("expected maxPerServer=5, got %d", p.maxPerServer)
	}
	if p.healthCheckInterval != 10*time.Second {
		t.Errorf("expected healthCheckInterval=10s, got %v", p.healthCheckInterval)
	}
	if p.idleTimeout != 1*time.Minute {
		t.Errorf("expected idleTimeout=1m, got %v", p.idleTimeout)
	}
}

func TestRegisterStdio(t *testing.T) {
	p := New()
	defer p.Close()

	err := p.RegisterStdio("test-server", "echo", []string{"hello"})
	if err != nil {
		t.Fatalf("RegisterStdio failed: %v", err)
	}

	servers := p.ListServers()
	if len(servers) != 1 {
		t.Errorf("expected 1 server, got %d", len(servers))
	}
	if servers[0] != "test-server" {
		t.Errorf("expected server name 'test-server', got '%s'", servers[0])
	}

	config, ok := p.ServerInfo("test-server")
	if !ok {
		t.Fatal("server not found")
	}
	if config.Type != ServerTypeStdio {
		t.Errorf("expected ServerTypeStdio, got %d", config.Type)
	}
	if config.Command != "echo" {
		t.Errorf("expected command 'echo', got '%s'", config.Command)
	}
}

func TestRegisterHTTP(t *testing.T) {
	p := New()
	defer p.Close()

	err := p.RegisterHTTP("http-server", "http://localhost:8080/mcp")
	if err != nil {
		t.Fatalf("RegisterHTTP failed: %v", err)
	}

	config, ok := p.ServerInfo("http-server")
	if !ok {
		t.Fatal("server not found")
	}
	if config.Type != ServerTypeHTTP {
		t.Errorf("expected ServerTypeHTTP, got %d", config.Type)
	}
	if config.URL != "http://localhost:8080/mcp" {
		t.Errorf("expected URL 'http://localhost:8080/mcp', got '%s'", config.URL)
	}
}

func TestRegisterInvalid(t *testing.T) {
	p := New()
	defer p.Close()

	tests := []struct {
		name    string
		fn      func() error
		wantErr bool
	}{
		{
			name:    "empty name stdio",
			fn:      func() error { return p.RegisterStdio("", "echo", nil) },
			wantErr: true,
		},
		{
			name:    "empty command stdio",
			fn:      func() error { return p.RegisterStdio("test", "", nil) },
			wantErr: true,
		},
		{
			name:    "empty name http",
			fn:      func() error { return p.RegisterHTTP("", "http://localhost") },
			wantErr: true,
		},
		{
			name:    "empty url http",
			fn:      func() error { return p.RegisterHTTP("test", "") },
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRegisterWithConfig(t *testing.T) {
	p := New()
	defer p.Close()

	config := ServerConfig{
		Name:           "custom-server",
		Type:           ServerTypeHTTP,
		URL:            "http://localhost:9090/mcp",
		MaxConnections: 3,
	}

	err := p.Register(config)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	retrieved, ok := p.ServerInfo("custom-server")
	if !ok {
		t.Fatal("server not found")
	}
	if retrieved.MaxConnections != 3 {
		t.Errorf("expected MaxConnections=3, got %d", retrieved.MaxConnections)
	}
}

func TestUnregister(t *testing.T) {
	p := New()
	defer p.Close()

	_ = p.RegisterHTTP("to-remove", "http://localhost:8080/mcp")

	if len(p.ListServers()) != 1 {
		t.Fatal("server not registered")
	}

	err := p.Unregister("to-remove")
	if err != nil {
		t.Fatalf("Unregister failed: %v", err)
	}

	if len(p.ListServers()) != 0 {
		t.Error("server not removed")
	}
}

func TestGetServerNotFound(t *testing.T) {
	p := New()
	defer p.Close()

	_, err := p.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent server")
	}
}

func TestPoolClosed(t *testing.T) {
	p := New()
	p.Close()

	err := p.RegisterStdio("test", "echo", nil)
	if err != ErrPoolClosed {
		t.Errorf("expected ErrPoolClosed, got %v", err)
	}

	err = p.RegisterHTTP("test", "http://localhost")
	if err != ErrPoolClosed {
		t.Errorf("expected ErrPoolClosed, got %v", err)
	}

	_, err = p.Get(context.Background(), "test")
	if err != ErrPoolClosed {
		t.Errorf("expected ErrPoolClosed, got %v", err)
	}
}

func TestDoubleClose(t *testing.T) {
	p := New()
	err := p.Close()
	if err != nil {
		t.Errorf("first close failed: %v", err)
	}

	err = p.Close()
	if err != ErrPoolClosed {
		t.Errorf("expected ErrPoolClosed on double close, got %v", err)
	}
}

func TestPoolStats(t *testing.T) {
	p := New()
	defer p.Close()

	_ = p.RegisterStdio("server1", "echo", nil)
	_ = p.RegisterHTTP("server2", "http://localhost:8080/mcp")

	stats := p.Stats()
	if stats.RegisteredServers != 2 {
		t.Errorf("expected 2 registered servers, got %d", stats.RegisteredServers)
	}
}

func TestConcurrentRegister(t *testing.T) {
	p := New()
	defer p.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "server-" + string(rune('a'+idx%26))
			_ = p.RegisterHTTP(name, "http://localhost:8080/mcp")
		}(i)
	}
	wg.Wait()

	// Should have at most 26 unique servers (a-z)
	if len(p.ListServers()) > 26 {
		t.Errorf("expected at most 26 servers, got %d", len(p.ListServers()))
	}
}
