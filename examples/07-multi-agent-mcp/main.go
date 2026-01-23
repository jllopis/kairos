// Package main runs the MCP remote agent example.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
	kmcp "github.com/jllopis/kairos/pkg/mcp"
)

// Example config (.kairos/settings.json):
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
// Run with mock provider (recommended for testing):
//
//	go run .
//
// Run with Ollama (requires ollama serve):
//
//	USE_OLLAMA=1 go run .
//
// Adjust timeout for slow models:
//
//	TIMEOUT_SECONDS=180 USE_OLLAMA=1 go run .
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

	// Use mock by default unless USE_OLLAMA=1
	useOllama := os.Getenv("USE_OLLAMA") == "1"
	var provider llm.Provider
	if useOllama {
		log.Println("using Ollama provider")
		provider = llm.NewOllama(cfg.LLM.BaseURL)
	} else {
		log.Println("using mock provider (set USE_OLLAMA=1 for real LLM)")
		mock := &llm.ScriptedMockProvider{}
		mock.AddResponse("I have access to the following tools: create_directory, list_directory, read_file, write_file, and more filesystem operations.\n\nFinal Answer: Tools listed successfully.")
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
		agent.WithMaxIterations(3), // Limit iterations for demo
	)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}
	defer func() {
		if err := ag.Close(); err != nil {
			log.Printf("failed to close agent: %v", err)
		}
	}()

	// List tools discovered via MCP
	log.Println("--- Discovering MCP tools ---")
	if tools, err := ag.MCPTools(ctx); err != nil {
		log.Printf("failed to list MCP tools: %v", err)
	} else {
		log.Printf("discovered %d tools from MCP servers", len(tools))
		for _, tool := range tools {
			log.Printf("  - %s: %s", tool.Name, truncate(tool.Description, 60))
		}
	}

	// Demonstrate direct MCP client usage
	log.Println("--- Direct MCP client demo ---")
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
		log.Printf("mcp %q: %d tools available", name, len(tools))

		// Call echo tool if available
		for _, tool := range tools {
			if tool.Name == "echo" {
				log.Printf("calling echo tool...")
				res, err := client.CallTool(ctx, "echo", map[string]interface{}{
					"message": "Hello from Kairos!",
				})
				if err != nil {
					log.Printf("tool call error: %v", err)
				} else if payload, err := json.MarshalIndent(res, "", "  "); err == nil {
					log.Printf("echo response: %s", payload)
				}
				break
			}
		}
		_ = client.Close()
	}

	// Run agent with timeout
	log.Println("--- Running agent ---")
	timeout := 60 * time.Second // Default timeout
	if useOllama {
		timeout = 120 * time.Second // Longer timeout for real LLM
	}
	if raw := os.Getenv("TIMEOUT_SECONDS"); raw != "" {
		if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
			timeout = time.Duration(secs) * time.Second
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	response, err := ag.Run(runCtx, "What tools do you have available?")
	if err != nil {
		log.Printf("agent run failed: %v", err)
		log.Println("Tip: Try TIMEOUT_SECONDS=180 for slow models, or USE_OLLAMA=0 for mock")
		return
	}
	fmt.Printf("\nAgent response:\n%v\n", response)
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
		return kmcp.NewClientWithStdioProtocol(server.Command, server.Args, server.Env, server.ProtocolVersion)
	case "http", "streamable-http", "streamablehttp":
		return kmcp.NewClientWithStreamableHTTPProtocol(server.URL, server.ProtocolVersion)
	default:
		return nil, fmt.Errorf("unsupported transport %q", server.Transport)
	}
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
