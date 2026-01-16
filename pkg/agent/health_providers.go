// Copyright 2026 © The Kairos Authors
// SPD:License-Identifier: Apache-2.0

// Package agent implements the LLM-driven agent loop and configuration options.
package agent

import (
	"context"
	"sync"
	"time"

	"github.com/jllopis/kairos/pkg/core"
)

// AgentHealthChecker provides health checks for agent components.
type AgentHealthChecker struct {
	agent       *Agent
	lastCheck   time.Time
	lastResult  core.HealthResult
	minInterval time.Duration
	mu          sync.RWMutex
}

// NewAgentHealthChecker creates a health checker for an agent.
func NewAgentHealthChecker(agent *Agent) *AgentHealthChecker {
	return &AgentHealthChecker{
		agent:       agent,
		minInterval: 5 * time.Second,
	}
}

// Check returns the health status of the agent.
func (h *AgentHealthChecker) Check(ctx context.Context) core.HealthResult {
	h.mu.RLock()
	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		result := h.lastResult
		h.mu.RUnlock()
		return result
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// Double-check after acquiring write lock
	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		return h.lastResult
	}

	result := core.HealthResult{
		Component: "agent:" + h.agent.id,
		LastCheck: time.Now(),
	}

	// Check if LLM provider is available
	if h.agent.llm == nil {
		result.Status = core.HealthUnhealthy
		result.Message = "LLM provider not configured"
		h.lastResult = result
		h.lastCheck = time.Now()
		return result
	}

	// Check MCP clients health
	mcpHealthy := true
	for _, client := range h.agent.mcpClients {
		if client == nil {
			mcpHealthy = false
			break
		}
	}

	if !mcpHealthy {
		result.Status = core.HealthDegraded
		result.Message = "some MCP clients unavailable"
	} else {
		result.Status = core.HealthHealthy
		result.Message = "agent operational"
	}

	h.lastResult = result
	h.lastCheck = time.Now()
	return result
}

// LLMHealthChecker provides health checks for LLM providers.
type LLMHealthChecker struct {
	name        string
	checkFunc   func(ctx context.Context) error
	lastCheck   time.Time
	lastResult  core.HealthResult
	minInterval time.Duration
	mu          sync.RWMutex
}

// NewLLMHealthChecker creates a health checker for an LLM provider.
func NewLLMHealthChecker(name string, checkFunc func(ctx context.Context) error) *LLMHealthChecker {
	return &LLMHealthChecker{
		name:        name,
		checkFunc:   checkFunc,
		minInterval: 30 * time.Second,
	}
}

// Check returns the health status of the LLM provider.
func (h *LLMHealthChecker) Check(ctx context.Context) core.HealthResult {
	h.mu.RLock()
	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		result := h.lastResult
		h.mu.RUnlock()
		return result
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		return h.lastResult
	}

	result := core.HealthResult{
		Component: "llm:" + h.name,
		LastCheck: time.Now(),
	}

	if h.checkFunc == nil {
		result.Status = core.HealthHealthy
		result.Message = "LLM provider available (no health check configured)"
		h.lastResult = result
		h.lastCheck = time.Now()
		return result
	}

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := h.checkFunc(checkCtx); err != nil {
		result.Status = core.HealthUnhealthy
		result.Message = err.Error()
		result.Error = err
	} else {
		result.Status = core.HealthHealthy
		result.Message = "LLM provider responsive"
	}

	h.lastResult = result
	h.lastCheck = time.Now()
	return result
}

// MemoryHealthChecker provides health checks for memory backends.
type MemoryHealthChecker struct {
	name        string
	memory      core.Memory
	lastCheck   time.Time
	lastResult  core.HealthResult
	minInterval time.Duration
	mu          sync.RWMutex
}

// NewMemoryHealthChecker creates a health checker for a memory backend.
func NewMemoryHealthChecker(name string, memory core.Memory) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		name:        name,
		memory:      memory,
		minInterval: 10 * time.Second,
	}
}

// Check returns the health status of the memory backend.
func (h *MemoryHealthChecker) Check(ctx context.Context) core.HealthResult {
	h.mu.RLock()
	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		result := h.lastResult
		h.mu.RUnlock()
		return result
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		return h.lastResult
	}

	result := core.HealthResult{
		Component: "memory:" + h.name,
		LastCheck: time.Now(),
	}

	if h.memory == nil {
		result.Status = core.HealthUnhealthy
		result.Message = "memory backend not configured"
		h.lastResult = result
		h.lastCheck = time.Now()
		return result
	}

	// Try a simple retrieve to check memory health
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := h.memory.Retrieve(checkCtx, nil)
	if err != nil {
		result.Status = core.HealthDegraded
		result.Message = "memory retrieve failed: " + err.Error()
		result.Error = err
	} else {
		result.Status = core.HealthHealthy
		result.Message = "memory backend responsive"
	}

	h.lastResult = result
	h.lastCheck = time.Now()
	return result
}

// MCPHealthChecker provides health checks for MCP clients.
type MCPHealthChecker struct {
	name        string
	listTools   func(ctx context.Context) (int, error)
	lastCheck   time.Time
	lastResult  core.HealthResult
	minInterval time.Duration
	mu          sync.RWMutex
}

// NewMCPHealthChecker creates a health checker for an MCP client.
func NewMCPHealthChecker(name string, listToolsFunc func(ctx context.Context) (int, error)) *MCPHealthChecker {
	return &MCPHealthChecker{
		name:        name,
		listTools:   listToolsFunc,
		minInterval: 30 * time.Second,
	}
}

// Check returns the health status of the MCP client.
func (h *MCPHealthChecker) Check(ctx context.Context) core.HealthResult {
	h.mu.RLock()
	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		result := h.lastResult
		h.mu.RUnlock()
		return result
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	if time.Since(h.lastCheck) < h.minInterval && !h.lastResult.LastCheck.IsZero() {
		return h.lastResult
	}

	result := core.HealthResult{
		Component: "mcp:" + h.name,
		LastCheck: time.Now(),
	}

	if h.listTools == nil {
		result.Status = core.HealthHealthy
		result.Message = "MCP client available (no health check configured)"
		h.lastResult = result
		h.lastCheck = time.Now()
		return result
	}

	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	toolCount, err := h.listTools(checkCtx)
	if err != nil {
		result.Status = core.HealthUnhealthy
		result.Message = "MCP tool discovery failed: " + err.Error()
		result.Error = err
	} else {
		result.Status = core.HealthHealthy
		result.Message = "MCP client operational"
		if toolCount > 0 {
			result.Message += " (" + string(rune('0'+toolCount%10)) + " tools)"
		}
	}

	h.lastResult = result
	h.lastCheck = time.Now()
	return result
}
