// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/governance"
	kairosmcp "github.com/jllopis/kairos/pkg/mcp"
	"github.com/jllopis/kairos/pkg/skills"
)

type validateResult struct {
	Config     checkResult   `json:"config"`
	LLM        checkResult   `json:"llm"`
	MCP        []checkResult `json:"mcp"`
	Governance checkResult   `json:"governance"`
	Skills     []checkResult `json:"skills"`
	Overall    string        `json:"overall"`
}

type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warn", "error", "skip"
	Message string `json:"message,omitempty"`
}

func runValidate(ctx context.Context, flags globalFlags, args []string) {
	result := validateResult{
		MCP:    []checkResult{},
		Skills: []checkResult{},
	}
	hasError := false
	hasWarn := false

	// 1. Validate config loading
	cfg, err := config.LoadWithCLI(flags.ConfigArgs)
	if err != nil {
		result.Config = checkResult{
			Name:    "config",
			Status:  "error",
			Message: fmt.Sprintf("failed to load: %v", err),
		}
		hasError = true
	} else {
		result.Config = checkResult{
			Name:   "config",
			Status: "ok",
		}
	}

	// 2. Validate LLM provider
	if cfg != nil {
		result.LLM = validateLLM(cfg)
		if result.LLM.Status == "error" {
			hasError = true
		} else if result.LLM.Status == "warn" {
			hasWarn = true
		}
	} else {
		result.LLM = checkResult{Name: "llm", Status: "skip", Message: "config not loaded"}
	}

	// 3. Validate MCP servers
	if cfg != nil && len(cfg.MCP.Servers) > 0 {
		result.MCP = validateMCPServers(ctx, cfg, flags.Timeout)
		for _, r := range result.MCP {
			if r.Status == "error" {
				hasError = true
			} else if r.Status == "warn" {
				hasWarn = true
			}
		}
	}

	// 4. Validate governance policies
	if cfg != nil {
		result.Governance = validateGovernance(cfg)
		if result.Governance.Status == "error" {
			hasError = true
		} else if result.Governance.Status == "warn" {
			hasWarn = true
		}
	} else {
		result.Governance = checkResult{Name: "governance", Status: "skip", Message: "config not loaded"}
	}

	// 5. Validate skills directories
	skillsDir := findSkillsDir()
	if skillsDir != "" {
		result.Skills = validateSkills(skillsDir)
		for _, r := range result.Skills {
			if r.Status == "error" {
				hasError = true
			} else if r.Status == "warn" {
				hasWarn = true
			}
		}
	}

	// Overall status
	if hasError {
		result.Overall = "error"
	} else if hasWarn {
		result.Overall = "warn"
	} else {
		result.Overall = "ok"
	}

	// Output
	if flags.JSON {
		printJSON(result)
		return
	}

	printValidateResult(result)

	if hasError {
		os.Exit(1)
	}
}

func validateLLM(cfg *config.Config) checkResult {
	if cfg.LLM.Provider == "" {
		return checkResult{
			Name:    "llm",
			Status:  "warn",
			Message: "no provider configured (will use default)",
		}
	}

	switch strings.ToLower(cfg.LLM.Provider) {
	case "ollama":
		baseURL := cfg.LLM.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		// Check if Ollama is reachable
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(baseURL + "/api/tags")
		if err != nil {
			return checkResult{
				Name:    "llm",
				Status:  "error",
				Message: fmt.Sprintf("ollama not reachable at %s: %v", baseURL, err),
			}
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return checkResult{
				Name:    "llm",
				Status:  "error",
				Message: fmt.Sprintf("ollama returned status %d", resp.StatusCode),
			}
		}
		model := cfg.LLM.Model
		if model == "" {
			return checkResult{
				Name:    "llm",
				Status:  "warn",
				Message: "ollama reachable but no model configured",
			}
		}
		return checkResult{
			Name:   "llm",
			Status: "ok",
			Message: fmt.Sprintf("ollama (%s)", model),
		}

	case "openai":
		if cfg.LLM.APIKey == "" {
			return checkResult{
				Name:    "llm",
				Status:  "error",
				Message: "openai configured but no api_key set",
			}
		}
		return checkResult{
			Name:   "llm",
			Status: "ok",
			Message: "openai (api key configured)",
		}

	case "mock":
		return checkResult{
			Name:    "llm",
			Status:  "ok",
			Message: "mock provider",
		}

	default:
		return checkResult{
			Name:    "llm",
			Status:  "warn",
			Message: fmt.Sprintf("unknown provider %q", cfg.LLM.Provider),
		}
	}
}

func validateMCPServers(ctx context.Context, cfg *config.Config, timeout time.Duration) []checkResult {
	results := make([]checkResult, 0, len(cfg.MCP.Servers))

	for name, server := range cfg.MCP.Servers {
		transport := strings.ToLower(strings.TrimSpace(server.Transport))
		if transport == "" {
			transport = "stdio"
		}

		switch transport {
		case "stdio":
			if strings.TrimSpace(server.Command) == "" {
				results = append(results, checkResult{
					Name:    fmt.Sprintf("mcp:%s", name),
					Status:  "error",
					Message: "missing command for stdio transport",
				})
				continue
			}
			// For stdio, we just validate config (starting process is expensive)
			results = append(results, checkResult{
				Name:    fmt.Sprintf("mcp:%s", name),
				Status:  "ok",
				Message: fmt.Sprintf("stdio: %s", server.Command),
			})

		case "http", "streamable-http", "streamablehttp":
			if server.URL == "" {
				results = append(results, checkResult{
					Name:    fmt.Sprintf("mcp:%s", name),
					Status:  "error",
					Message: "missing url for http transport",
				})
				continue
			}
			// Try to connect
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			client, err := kairosmcp.NewClientWithStreamableHTTPProtocol(server.URL, server.ProtocolVersion)
			if err != nil {
				cancel()
				results = append(results, checkResult{
					Name:    fmt.Sprintf("mcp:%s", name),
					Status:  "error",
					Message: fmt.Sprintf("failed to connect: %v", err),
				})
				continue
			}
			tools, err := client.ListTools(checkCtx)
			cancel()
			_ = client.Close()
			if err != nil {
				results = append(results, checkResult{
					Name:    fmt.Sprintf("mcp:%s", name),
					Status:  "error",
					Message: fmt.Sprintf("failed to list tools: %v", err),
				})
				continue
			}
			results = append(results, checkResult{
				Name:    fmt.Sprintf("mcp:%s", name),
				Status:  "ok",
				Message: fmt.Sprintf("http: %d tools available", len(tools)),
			})

		default:
			results = append(results, checkResult{
				Name:    fmt.Sprintf("mcp:%s", name),
				Status:  "error",
				Message: fmt.Sprintf("unsupported transport %q", transport),
			})
		}
	}

	return results
}

func validateGovernance(cfg *config.Config) checkResult {
	if len(cfg.Governance.Policies) == 0 {
		return checkResult{
			Name:    "governance",
			Status:  "ok",
			Message: "no policies configured (default: allow all)",
		}
	}

	// Validate each policy
	for i, policy := range cfg.Governance.Policies {
		if policy.Effect == "" {
			return checkResult{
				Name:    "governance",
				Status:  "error",
				Message: fmt.Sprintf("policy %d: missing effect", i),
			}
		}
		effect := strings.ToLower(policy.Effect)
		if effect != "allow" && effect != "deny" && effect != "pending" {
			return checkResult{
				Name:    "governance",
				Status:  "error",
				Message: fmt.Sprintf("policy %d: invalid effect %q (must be allow/deny/pending)", i, policy.Effect),
			}
		}
	}

	// Try to build the ruleset
	ruleset := governance.RuleSetFromConfig(cfg.Governance)
	if ruleset == nil {
		return checkResult{
			Name:    "governance",
			Status:  "error",
			Message: "failed to build ruleset",
		}
	}

	return checkResult{
		Name:    "governance",
		Status:  "ok",
		Message: fmt.Sprintf("%d policies configured", len(cfg.Governance.Policies)),
	}
}

func validateSkills(dir string) []checkResult {
	skillSpecs, err := skills.LoadDir(dir)
	if err != nil {
		return []checkResult{{
			Name:    "skills",
			Status:  "error",
			Message: fmt.Sprintf("failed to load skills from %s: %v", dir, err),
		}}
	}

	if len(skillSpecs) == 0 {
		return []checkResult{{
			Name:    "skills",
			Status:  "ok",
			Message: fmt.Sprintf("no skills found in %s", dir),
		}}
	}

	results := make([]checkResult, 0, len(skillSpecs))
	for _, skill := range skillSpecs {
		results = append(results, checkResult{
			Name:    fmt.Sprintf("skill:%s", skill.Name),
			Status:  "ok",
			Message: truncateString(skill.Description, 50),
		})
	}

	return results
}

func findSkillsDir() string {
	// Check common locations
	candidates := []string{
		"./skills",
		"./internal/skills",
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}

func truncateString(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func printValidateResult(result validateResult) {
	statusIcon := map[string]string{
		"ok":    "✓",
		"warn":  "⚠",
		"error": "✗",
		"skip":  "○",
	}

	fmt.Println("Kairos Configuration Validation")
	fmt.Println("================================")
	fmt.Println()

	// Config
	printCheck(statusIcon, result.Config)

	// LLM
	printCheck(statusIcon, result.LLM)

	// MCP
	if len(result.MCP) > 0 {
		for _, r := range result.MCP {
			printCheck(statusIcon, r)
		}
	} else {
		fmt.Printf("%s mcp: no servers configured\n", statusIcon["ok"])
	}

	// Governance
	printCheck(statusIcon, result.Governance)

	// Skills
	if len(result.Skills) > 0 {
		for _, r := range result.Skills {
			printCheck(statusIcon, r)
		}
	}

	fmt.Println()
	switch result.Overall {
	case "ok":
		fmt.Println("✓ All checks passed")
	case "warn":
		fmt.Println("⚠ Validation completed with warnings")
	case "error":
		fmt.Println("✗ Validation failed")
	}
}

func printCheck(icons map[string]string, r checkResult) {
	icon := icons[r.Status]
	if r.Message != "" {
		fmt.Printf("%s %s: %s\n", icon, r.Name, r.Message)
	} else {
		fmt.Printf("%s %s\n", icon, r.Name)
	}
}
