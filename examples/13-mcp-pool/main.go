// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Example: MCP Connection Pool for Multi-Agent Scenarios
//
// This example demonstrates how to use the MCP connection pool to share
// connections across multiple agents efficiently.
//
// Key concepts:
//   - Creating a shared MCP connection pool
//   - Registering MCP servers (stdio and HTTP)
//   - Multiple agents sharing the same connections
//   - Proper resource cleanup with reference counting
//
// Run: go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jllopis/kairos/pkg/mcp/pool"
)

func main() {
	ctx := context.Background()

	// Create a shared MCP connection pool
	// This pool can be used across all agents in your application
	mcpPool := pool.New(
		pool.WithMaxConnectionsPerServer(5),    // Max 5 connections per MCP server
		pool.WithHealthCheckInterval(30*time.Second), // Check health every 30s
		pool.WithIdleTimeout(5*time.Minute),    // Close idle connections after 5m
	)
	defer mcpPool.Close()

	// Register MCP servers that will be shared
	// In a real application, you might load these from configuration

	// Example 1: Register a stdio-based MCP server
	// This would start the npx process when first connection is requested
	if err := mcpPool.RegisterStdio(
		"filesystem",
		"npx",
		[]string{"-y", "@modelcontextprotocol/server-filesystem", "/tmp"},
	); err != nil {
		log.Fatalf("Failed to register filesystem server: %v", err)
	}

	// Example 2: Register an HTTP-based MCP server
	// This connects to an already-running server
	if err := mcpPool.RegisterHTTP(
		"github",
		"http://localhost:8080/mcp",
	); err != nil {
		log.Fatalf("Failed to register github server: %v", err)
	}

	// Show registered servers
	fmt.Println("Registered MCP servers:", mcpPool.ListServers())

	// Simulate multiple agents using the pool concurrently
	// In a real application, each goroutine would be an agent
	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(agentID int) {
			defer wg.Done()
			simulateAgent(ctx, mcpPool, agentID)
		}(i)
	}

	wg.Wait()

	// Print final statistics
	stats := mcpPool.Stats()
	fmt.Println("\n=== Pool Statistics ===")
	fmt.Printf("Registered Servers:  %d\n", stats.RegisteredServers)
	fmt.Printf("Active Connections:  %d\n", stats.ActiveConnections)
	fmt.Printf("Total Connections:   %d\n", stats.TotalConnections)
	fmt.Printf("Connection Errors:   %d\n", stats.ConnectionErrors)
	fmt.Printf("Health Checks OK:    %d\n", stats.HealthChecksPassed)
	fmt.Printf("Health Checks Failed: %d\n", stats.HealthChecksFailed)
}

func simulateAgent(ctx context.Context, mcpPool *pool.Pool, agentID int) {
	fmt.Printf("[Agent %d] Starting\n", agentID)

	// Agent requests a connection from the pool
	// Multiple agents asking for the same server will share connections
	client, err := mcpPool.Get(ctx, "filesystem")
	if err != nil {
		// In this example, the MCP server isn't actually running
		// so we expect an error. In production, you'd handle this gracefully.
		fmt.Printf("[Agent %d] Could not get filesystem client (expected in demo): %v\n", agentID, err)
		return
	}

	// Use the client
	fmt.Printf("[Agent %d] Got filesystem client, listing tools...\n", agentID)

	tools, err := client.ListTools(ctx)
	if err != nil {
		fmt.Printf("[Agent %d] Error listing tools: %v\n", agentID, err)
	} else {
		fmt.Printf("[Agent %d] Found %d tools\n", agentID, len(tools))
	}

	// Simulate some work
	time.Sleep(100 * time.Millisecond)

	// Release the connection back to the pool
	// The connection is not closed, just marked as available for reuse
	mcpPool.Release("filesystem", client)
	fmt.Printf("[Agent %d] Released connection\n", agentID)
}
