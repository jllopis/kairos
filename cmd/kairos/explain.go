// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/jllopis/kairos/pkg/config"
	kairosmcp "github.com/jllopis/kairos/pkg/mcp"
	"github.com/jllopis/kairos/pkg/skills"
)

type explainResult struct {
	Agent      string            `json:"agent"`
	Role       string            `json:"role,omitempty"`
	LLM        explainLLM        `json:"llm"`
	Memory     explainMemory     `json:"memory"`
	Governance explainGovernance `json:"governance"`
	Tools      []explainTool     `json:"tools"`
	Skills     []explainSkill    `json:"skills"`
	A2A        explainA2A        `json:"a2a"`
}

type explainLLM struct {
	Provider string `json:"provider"`
	Model    string `json:"model,omitempty"`
	BaseURL  string `json:"base_url,omitempty"`
}

type explainMemory struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

type explainGovernance struct {
	Enabled  bool     `json:"enabled"`
	Policies []string `json:"policies,omitempty"`
}

type explainTool struct {
	Name        string `json:"name"`
	Source      string `json:"source"`
	Description string `json:"description,omitempty"`
}

type explainSkill struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type explainA2A struct {
	Enabled bool `json:"enabled"`
}

func runExplain(ctx context.Context, global globalFlags, args []string) {
	fs := flag.NewFlagSet("explain", flag.ExitOnError)
	agentID := fs.String("agent", "kairos-agent", "Agent ID to explain")
	skillsDir := fs.String("skills", "", "Skills directory path")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}

	cfg, err := config.LoadWithCLI(global.ConfigArgs)
	if err != nil {
		fatal(err)
	}

	result := buildExplainResult(ctx, cfg, *agentID, *skillsDir)

	if global.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fatal(err)
		}
		return
	}

	printExplainTree(result)
}

func buildExplainResult(ctx context.Context, cfg *config.Config, agentID, skillsDir string) explainResult {
	result := explainResult{
		Agent: agentID,
	}

	// LLM
	result.LLM = explainLLM{
		Provider: cfg.LLM.Provider,
		Model:    cfg.LLM.Model,
		BaseURL:  cfg.LLM.BaseURL,
	}
	if result.LLM.Provider == "" {
		result.LLM.Provider = "not configured"
	}

	// Memory
	result.Memory = explainMemory{
		Enabled:  cfg.Memory.Enabled,
		Provider: cfg.Memory.Provider,
	}
	if result.Memory.Provider == "" {
		result.Memory.Provider = "inmemory"
	}

	// Governance
	result.Governance = explainGovernance{
		Enabled:  len(cfg.Governance.Policies) > 0,
		Policies: extractPolicyNames(cfg.Governance.Policies),
	}

	// Tools from MCP servers
	result.Tools = collectTools(ctx, cfg)

	// Skills
	if skillsDir != "" {
		result.Skills = collectSkills(skillsDir)
	}

	// A2A - check if any agents have A2A config
	result.A2A = explainA2A{
		Enabled: len(cfg.Agents) > 0,
	}

	// Agent-specific config overrides
	if _, ok := cfg.Agents[agentID]; ok {
		result.A2A.Enabled = true
	}

	return result
}

func extractPolicyNames(policies []config.PolicyRuleConfig) []string {
	names := make([]string, 0, len(policies))
	for _, p := range policies {
		if p.ID != "" {
			names = append(names, p.ID)
		}
	}
	return names
}

func collectTools(ctx context.Context, cfg *config.Config) []explainTool {
	tools := make([]explainTool, 0)

	for name, serverCfg := range cfg.MCP.Servers {
		mcpTools, err := listMCPTools(ctx, serverCfg, name)
		if err != nil {
			tools = append(tools, explainTool{
				Name:        fmt.Sprintf("error: %s", name),
				Source:      fmt.Sprintf("MCP: %s", name),
				Description: err.Error(),
			})
			continue
		}
		for _, t := range mcpTools {
			tools = append(tools, explainTool{
				Name:        t.Name,
				Source:      fmt.Sprintf("MCP: %s", name),
				Description: t.Description,
			})
		}
	}

	return tools
}

func listMCPTools(ctx context.Context, serverCfg config.MCPServerConfig, serverName string) ([]explainTool, error) {
	var client *kairosmcp.Client
	var err error

	switch serverCfg.Transport {
	case "http":
		if serverCfg.URL == "" {
			return nil, fmt.Errorf("no URL configured")
		}
		client, err = kairosmcp.NewClientWithStreamableHTTP(serverCfg.URL)
	case "stdio":
		if serverCfg.Command == "" {
			return nil, fmt.Errorf("no command configured")
		}
		client, err = kairosmcp.NewClientWithStdio(serverCfg.Command, serverCfg.Args, serverCfg.Env)
	default:
		return nil, fmt.Errorf("unsupported transport: %s", serverCfg.Transport)
	}

	if err != nil {
		return nil, err
	}
	defer client.Close()

	mcpTools, err := client.ListTools(ctx)
	if err != nil {
		return nil, err
	}

	tools := make([]explainTool, 0, len(mcpTools))
	for _, t := range mcpTools {
		tools = append(tools, explainTool{
			Name:        t.Name,
			Description: t.Description,
		})
	}
	return tools, nil
}

func collectSkills(dir string) []explainSkill {
	specs, err := skills.LoadDir(dir)
	if err != nil {
		return []explainSkill{{Name: "error", Description: err.Error()}}
	}

	result := make([]explainSkill, 0, len(specs))
	for _, s := range specs {
		result = append(result, explainSkill{
			Name:        s.Name,
			Description: truncate(s.Description, 60),
		})
	}
	return result
}

func printExplainTree(r explainResult) {
	fmt.Printf("Agent: %s\n", r.Agent)
	if r.Role != "" {
		fmt.Printf("│   Role: %s\n", r.Role)
	}

	// LLM
	llmInfo := r.LLM.Provider
	if r.LLM.Model != "" {
		llmInfo = fmt.Sprintf("%s (%s)", r.LLM.Provider, r.LLM.Model)
	}
	fmt.Printf("├── LLM: %s\n", llmInfo)

	// Memory
	memInfo := r.Memory.Provider
	if !r.Memory.Enabled {
		memInfo = "disabled"
	}
	fmt.Printf("├── Memory: %s\n", memInfo)

	// Governance
	if r.Governance.Enabled {
		fmt.Printf("├── Governance: enabled\n")
		for i, p := range r.Governance.Policies {
			prefix := "│   ├──"
			if i == len(r.Governance.Policies)-1 {
				prefix = "│   └──"
			}
			fmt.Printf("%s Policy: %s\n", prefix, p)
		}
	} else {
		fmt.Printf("├── Governance: disabled\n")
	}

	// Tools
	fmt.Printf("├── Tools: %d\n", len(r.Tools))
	for i, t := range r.Tools {
		prefix := "│   ├──"
		if i == len(r.Tools)-1 {
			prefix = "│   └──"
		}
		fmt.Printf("%s %s (%s)\n", prefix, t.Name, t.Source)
	}

	// Skills
	if len(r.Skills) > 0 {
		fmt.Printf("├── Skills: %d\n", len(r.Skills))
		for i, s := range r.Skills {
			prefix := "│   ├──"
			if i == len(r.Skills)-1 {
				prefix = "│   └──"
			}
			if s.Description != "" {
				fmt.Printf("%s %s: %s\n", prefix, s.Name, s.Description)
			} else {
				fmt.Printf("%s %s\n", prefix, s.Name)
			}
		}
	}

	// A2A
	if r.A2A.Enabled {
		fmt.Printf("└── A2A: enabled\n")
	} else {
		fmt.Printf("└── A2A: disabled\n")
	}
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
