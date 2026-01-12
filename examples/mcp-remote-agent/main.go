// Package main runs the MCP remote agent example.
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
	kmcp "github.com/jllopis/kairos/pkg/mcp"
)

// Example config (settings.json):
//
//	{
//	  "mcp": {
//	    "servers": {
//	      "example-mcp": {
//	        "transport": "http",
//	        "url": "http://localhost:8080/mcp"
//	      }
//	    }
//	  }
//	}
//
// Then run:
//
//	go run ./examples/mcp-remote-agent
func main() {
	ctx := context.Background()

	cfg, err := config.LoadWithCLI(os.Args[1:])
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

	agentCfg := cfg.AgentConfigFor("remote-mcp-agent")
	ag, err := agent.New("remote-mcp-agent", provider,
		agent.WithRole("Remote MCP client"),
		agent.WithSkills([]core.Skill{}),
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

	for name, server := range cfg.MCP.Servers {
		client, err := newMCPClient(server)
		if err != nil {
			log.Printf("mcp client %q: %v", name, err)
			continue
		}
		tools, err := client.ListTools(ctx)
		if err != nil {
			log.Printf("mcp list tools %q: %v", name, err)
			_ = client.Close()
			continue
		}
		log.Printf("mcp %q tools=%d", name, len(tools))
		if len(tools) == 0 {
			_ = client.Close()
			continue
		}
		tool := tools[0]
		log.Printf("calling tool=%s (empty args)", tool.Name)
		res, err := client.CallTool(ctx, tool.Name, map[string]interface{}{})
		if err != nil {
			log.Printf("tool call error: %v", err)
		} else if payload, err := json.MarshalIndent(res, "", "  "); err == nil {
			log.Printf("tool response: %s", payload)
		}
		_ = client.Close()
	}

	response, err := ag.Run(ctx, "List the tools you can use.")
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}
	fmt.Printf("Agent response: %v\n", response)
}

func newMCPClient(server config.MCPServerConfig) (*kmcp.Client, error) {
	transport := strings.ToLower(strings.TrimSpace(server.Transport))
	if transport == "" {
		transport = "stdio"
	}
	switch transport {
	case "stdio":
		if strings.TrimSpace(server.Command) == "" {
			return nil, fmt.Errorf("missing command for stdio transport")
		}
		return kmcp.NewClientWithStdioProtocol(server.Command, server.Args, server.ProtocolVersion)
	case "http", "streamable-http", "streamablehttp":
		return kmcp.NewClientWithStreamableHTTPProtocol(server.URL, server.ProtocolVersion)
	default:
		return nil, fmt.Errorf("unsupported transport %q", server.Transport)
	}
}
