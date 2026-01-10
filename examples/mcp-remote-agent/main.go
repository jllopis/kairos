package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// Example config (settings.json):
//
//	{
//	  "mcpServers": {
//	    "microsoft.docs.mcp": {
//	      "type": "http",
//	      "url": "https://learn.microsoft.com/api/mcp"
//	    }
//	  }
//	}
//
// Then run:
//
//	go run ./examples/mcp-remote-agent
func main() {
	ctx := context.Background()

	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if len(cfg.MCP.Servers) == 0 {
		log.Fatal("no MCP servers configured (see example config in comments)")
	}

	for name, server := range cfg.MCP.Servers {
		log.Printf("configured MCP server: %s (%s)", name, server.Transport)
	}

	var provider llm.Provider
	switch cfg.LLM.Provider {
	case "ollama":
		provider = llm.NewOllama(cfg.LLM.BaseURL)
	default:
		mock := &llm.ScriptedMockProvider{}
		mock.AddResponse("Final Answer: listo")
		provider = mock
	}

	ag, err := agent.New("remote-mcp-agent", provider,
		agent.WithRole("Remote MCP client"),
		agent.WithSkills([]core.Skill{}),
		agent.WithModel(cfg.LLM.Model),
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

	response, err := ag.Run(ctx, "List the tools you can use.")
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}
	fmt.Printf("Agent response: %v\n", response)
}
