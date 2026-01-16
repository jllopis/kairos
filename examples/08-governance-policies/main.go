// Package main demonstrates policy-based denial of a real MCP call.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/governance"
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
//	  },
//	  "governance": {
//	    "policies": [
//	      {
//	        "id": "deny-tools",
//	        "effect": "deny",
//	        "type": "tool",
//	        "name": "*",
//	        "reason": "blocked by policy"
//	      }
//	    ]
//	  }
//	}
//
// Then run:
//
//	go run ./examples/mcp-remote-policy-forbid
func main() {
	ctx := context.Background()

	cfg, err := config.LoadWithCLI(os.Args[1:])
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if len(cfg.MCP.Servers) == 0 {
		log.Fatal("no MCP servers configured (see example config in comments)")
	}

	policy := governance.RuleSetFromConfig(cfg.Governance)

	for name, server := range cfg.MCP.Servers {
		client, err := newMCPClient(server, policy, name)
		if err != nil {
			log.Fatalf("mcp client %q: %v", name, err)
		}
		tools, err := client.ListTools(ctx)
		if err != nil {
			log.Fatalf("list tools: %v", err)
		}
		if len(tools) == 0 {
			log.Printf("no tools found for %q", name)
			continue
		}

		tool := tools[0]
		log.Printf("attempting tool call (should be denied): %s", tool.Name)
		_, err = client.CallTool(ctx, tool.Name, map[string]interface{}{})
		if err != nil {
			log.Printf("policy denied tool call: %v", err)
		} else {
			log.Printf("unexpected: tool call allowed")
		}
		_ = client.Close()
	}
}

func newMCPClient(server config.MCPServerConfig, policy governance.PolicyEngine, name string) (*kmcp.Client, error) {
	transport := strings.ToLower(strings.TrimSpace(server.Transport))
	if transport == "" {
		transport = "stdio"
	}
	opts := []kmcp.ClientOption{
		kmcp.WithPolicyEngine(policy),
		kmcp.WithServerName(name),
	}
	switch transport {
	case "stdio":
		if strings.TrimSpace(server.Command) == "" {
			return nil, fmt.Errorf("missing command for stdio transport")
		}
		return kmcp.NewClientWithStdioProtocol(server.Command, server.Args, server.ProtocolVersion, opts...)
	case "http", "streamable-http", "streamablehttp":
		return kmcp.NewClientWithStreamableHTTPProtocol(server.URL, server.ProtocolVersion, opts...)
	default:
		return nil, fmt.Errorf("unsupported transport %q", server.Transport)
	}
}
