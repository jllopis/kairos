// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
)

// Adapter describes an available provider/backend in Kairos.
type Adapter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	ConfigKeys  []string `json:"config_keys,omitempty"`
	Docs        string   `json:"docs,omitempty"`
}

// adaptersRegistry is the catalog of known adapters.
var adaptersRegistry = []Adapter{
	// LLM Providers
	{
		Name:        "ollama",
		Type:        "llm",
		Description: "Local LLM inference with Ollama",
		ConfigKeys:  []string{"llm.provider=ollama", "llm.base_url", "llm.model"},
		Docs:        "https://ollama.ai",
	},
	{
		Name:        "openai",
		Type:        "llm",
		Description: "OpenAI GPT models (GPT-4, GPT-3.5)",
		ConfigKeys:  []string{"llm.provider=openai", "llm.api_key", "llm.model"},
		Docs:        "https://platform.openai.com/docs",
	},
	{
		Name:        "anthropic",
		Type:        "llm",
		Description: "Anthropic Claude models",
		ConfigKeys:  []string{"llm.provider=anthropic", "llm.api_key", "llm.model"},
		Docs:        "https://docs.anthropic.com",
	},
	{
		Name:        "mock",
		Type:        "llm",
		Description: "Mock LLM for testing (returns canned responses)",
		ConfigKeys:  []string{"llm.provider=mock"},
		Docs:        "pkg/llm/mock.go",
	},

	// Memory Backends
	{
		Name:        "inmemory",
		Type:        "memory",
		Description: "In-memory storage (non-persistent)",
		ConfigKeys:  []string{"memory.backend=inmemory"},
		Docs:        "pkg/memory/inmemory.go",
	},
	{
		Name:        "file",
		Type:        "memory",
		Description: "File-based persistent storage",
		ConfigKeys:  []string{"memory.backend=file", "memory.path"},
		Docs:        "pkg/memory/file.go",
	},
	{
		Name:        "vector",
		Type:        "memory",
		Description: "Vector store for semantic search",
		ConfigKeys:  []string{"memory.backend=vector", "memory.embeddings_url"},
		Docs:        "pkg/memory/vector.go",
	},

	// MCP Transports
	{
		Name:        "mcp-stdio",
		Type:        "mcp",
		Description: "MCP server via stdio (subprocess)",
		ConfigKeys:  []string{"mcp.servers.<name>.transport=stdio", "mcp.servers.<name>.command"},
		Docs:        "https://modelcontextprotocol.io",
	},
	{
		Name:        "mcp-http",
		Type:        "mcp",
		Description: "MCP server via HTTP",
		ConfigKeys:  []string{"mcp.servers.<name>.transport=http", "mcp.servers.<name>.url"},
		Docs:        "https://modelcontextprotocol.io",
	},

	// A2A
	{
		Name:        "a2a-grpc",
		Type:        "a2a",
		Description: "Agent-to-Agent communication via gRPC",
		ConfigKeys:  []string{"a2a.enabled=true", "a2a.grpc_addr"},
		Docs:        "docs/protocols/A2A/",
	},
	{
		Name:        "a2a-http",
		Type:        "a2a",
		Description: "Agent-to-Agent communication via HTTP+JSON",
		ConfigKeys:  []string{"a2a.enabled=true", "a2a.http_addr"},
		Docs:        "docs/protocols/A2A/",
	},

	// Telemetry
	{
		Name:        "otel-stdout",
		Type:        "telemetry",
		Description: "OpenTelemetry export to stdout",
		ConfigKeys:  []string{"telemetry.exporter=stdout"},
		Docs:        "docs/OBSERVABILITY.md",
	},
	{
		Name:        "otel-otlp",
		Type:        "telemetry",
		Description: "OpenTelemetry export via OTLP",
		ConfigKeys:  []string{"telemetry.exporter=otlp", "telemetry.otlp_endpoint"},
		Docs:        "docs/OBSERVABILITY.md",
	},
}

type adaptersListResult struct {
	Adapters []Adapter `json:"adapters"`
	Total    int       `json:"total"`
}

type adapterInfoResult struct {
	Adapter Adapter `json:"adapter"`
	Found   bool    `json:"found"`
}

func runAdapters(global globalFlags, args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("usage: kairos adapters <list|info> [args]"))
	}

	switch args[0] {
	case "list":
		runAdaptersList(global, args[1:])
	case "info":
		runAdaptersInfo(global, args[1:])
	default:
		fatal(fmt.Errorf("unknown adapters subcommand %q; use list or info", args[0]))
	}
}

func runAdaptersList(global globalFlags, args []string) {
	fs := flag.NewFlagSet("adapters list", flag.ExitOnError)
	filterType := fs.String("type", "", "Filter by type: llm, memory, mcp, a2a, telemetry")
	if err := fs.Parse(args); err != nil {
		fatal(err)
	}

	adapters := adaptersRegistry
	if *filterType != "" {
		filtered := make([]Adapter, 0)
		for _, a := range adapters {
			if a.Type == *filterType {
				filtered = append(filtered, a)
			}
		}
		adapters = filtered
	}

	result := adaptersListResult{
		Adapters: adapters,
		Total:    len(adapters),
	}

	if global.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fatal(err)
		}
		return
	}

	if len(adapters) == 0 {
		fmt.Println("No adapters found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tTYPE\tDESCRIPTION")
	fmt.Fprintln(w, "----\t----\t-----------")
	for _, a := range adapters {
		fmt.Fprintf(w, "%s\t%s\t%s\n", a.Name, a.Type, a.Description)
	}
	w.Flush()

	fmt.Printf("\nTotal: %d adapters\n", result.Total)
	fmt.Println("\nUse 'kairos adapters info <name>' for configuration details.")
}

func runAdaptersInfo(global globalFlags, args []string) {
	if len(args) == 0 {
		fatal(fmt.Errorf("usage: kairos adapters info <adapter-name>"))
	}

	name := args[0]
	var found *Adapter
	for _, a := range adaptersRegistry {
		if a.Name == name {
			found = &a
			break
		}
	}

	result := adapterInfoResult{
		Found: found != nil,
	}
	if found != nil {
		result.Adapter = *found
	}

	if global.JSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fatal(err)
		}
		return
	}

	if found == nil {
		fmt.Printf("Adapter %q not found.\n", name)
		fmt.Println("\nAvailable adapters:")
		for _, a := range adaptersRegistry {
			fmt.Printf("  - %s (%s)\n", a.Name, a.Type)
		}
		os.Exit(1)
	}

	fmt.Printf("Adapter: %s\n", found.Name)
	fmt.Printf("Type: %s\n", found.Type)
	fmt.Printf("Description: %s\n", found.Description)
	fmt.Println()

	if len(found.ConfigKeys) > 0 {
		fmt.Println("Configuration:")
		for _, k := range found.ConfigKeys {
			fmt.Printf("  • %s\n", k)
		}
		fmt.Println()
	}

	if found.Docs != "" {
		fmt.Printf("Documentation: %s\n", found.Docs)
	}
}
