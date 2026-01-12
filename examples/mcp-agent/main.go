// Package main runs the MCP tool-enabled agent example.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// Example usage with MCP configuration from kairos.json.
// Create one of:
//
//	./.kairos/settings.json
//	$HOME/.kairos/settings.json
//	$XDG_CONFIG_HOME/kairos/settings.json
//
// Example content:
//
//	{
//	  "mcpServers": {
//	    "fetch-stdio": {
//	      "transport": "stdio",
//	      "command": "docker",
//	      "args": ["run", "-i", "--rm", "mcp/fetch"]
//	    },
//	    "fetch-http": {
//	      "transport": "http",
//	      "url": "http://localhost:8080/mcp"
//	    }
//	  }
//	}
//
// Then run:
//
//	go run ./examples/mcp-agent
func main() {
	ctx := context.Background()

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if len(cfg.MCP.Servers) == 0 {
		log.Fatal("no MCP servers configured (see example config in comments)")
	}
	for name := range cfg.MCP.Servers {
		log.Printf("configured MCP server: %s", name)
	}

	useOllama := os.Getenv("USE_OLLAMA") != "0"
	var provider llm.Provider
	if useOllama {
		provider = llm.NewOllama(cfg.LLM.BaseURL)
	} else {
		mock := &llm.ScriptedMockProvider{}
		mock.AddResponse("Thought: need fetch. Action: fetch\nAction Input: {\"url\":\"https://example.com\"}")
		mock.AddResponse("Final Answer: listo")
		provider = mock
	}

	agentCfg := cfg.AgentConfigFor("mcp-agent")
	ag, err := agent.New("mcp-agent", provider,
		agent.WithRole("Tool user"),
		agent.WithSkills([]core.Skill{{Name: "fetch"}}),
		agent.WithModel(cfg.LLM.Model),
		agent.WithDisableActionFallback(agentCfg.DisableActionFallback),
		agent.WithActionFallbackWarning(agentCfg.WarnOnActionFallback),
		agent.WithMCPServerConfigs(cfg.MCP.Servers),
	)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}
	defer func() {
		if err := ag.Close(); err != nil {
			log.Printf("failed to close agent: %v", err)
		}
	}()

	if tools, err := ag.MCPTools(ctx); err != nil {
		log.Printf("failed to list MCP tools: %v", err)
	} else {
		for _, tool := range tools {
			payload, err := json.MarshalIndent(tool, "", "  ")
			if err != nil {
				log.Printf("failed to marshal tool definition (%s): %v", tool.Name, err)
				continue
			}
			log.Printf("tool definition: %s", string(payload))
		}
	}

	response, err := ag.Run(ctx, "Fetch https://example.com")
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	fmt.Printf("Agent response: %v\n", response)

	toolNames := ag.ToolNames()
	if len(toolNames) > 0 {
		log.Printf("discovered tools: %s", strings.Join(toolNames, ", "))
	}
}
