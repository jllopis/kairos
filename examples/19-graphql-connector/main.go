// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Example 19: GraphQL Connector
//
// This example demonstrates how to use the GraphQL connector to automatically
// generate tools from a GraphQL schema via introspection.
//
// The connector:
// 1. Connects to a GraphQL endpoint
// 2. Performs introspection to discover queries and mutations
// 3. Generates llm.Tool for each operation
// 4. Executes operations when called
//
// Usage:
//
//	export GRAPHQL_ENDPOINT="https://countries.trevorblades.com/graphql"
//	go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jllopis/kairos/pkg/connectors"
)

func main() {
	// Get GraphQL endpoint from environment or use a public API
	endpoint := os.Getenv("GRAPHQL_ENDPOINT")
	if endpoint == "" {
		// Use the Countries GraphQL API (public, no auth required)
		endpoint = "https://countries.trevorblades.com/graphql"
	}

	fmt.Println("🔗 GraphQL Connector Example")
	fmt.Println("============================")
	fmt.Printf("Endpoint: %s\n\n", endpoint)

	// Create the connector with introspection
	fmt.Println("📡 Performing introspection...")
	connector, err := connectors.NewGraphQLConnector(endpoint)
	if err != nil {
		log.Fatalf("Failed to create connector: %v", err)
	}

	// Get generated tools
	tools := connector.Tools()
	fmt.Printf("\n✅ Generated %d tools from schema:\n\n", len(tools))

	// Display tools
	for i, tool := range tools {
		if i >= 10 {
			fmt.Printf("   ... and %d more\n", len(tools)-10)
			break
		}
		fmt.Printf("   📦 %s\n", tool.Function.Name)
		if tool.Function.Description != "" {
			desc := tool.Function.Description
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			fmt.Printf("      %s\n", desc)
		}

		// Show parameters
		if params, ok := tool.Function.Parameters.(map[string]interface{}); ok {
			if props, ok := params["properties"].(map[string]interface{}); ok && len(props) > 0 {
				fmt.Printf("      Args: ")
				first := true
				for name := range props {
					if !first {
						fmt.Print(", ")
					}
					fmt.Print(name)
					first = false
				}
				fmt.Println()
			}
		}
	}

	// Execute a query if this is the countries API
	if endpoint == "https://countries.trevorblades.com/graphql" {
		fmt.Println("\n📊 Executing sample query...")
		ctx := context.Background()

		// Try to find and execute the 'countries' query
		for _, tool := range tools {
			if tool.Function.Name == "countries" || tool.Function.Name == "country" {
				fmt.Printf("\n   Calling: %s\n", tool.Function.Name)

				var result interface{}
				var err error

				if tool.Function.Name == "country" {
					// Get a specific country
					result, err = connector.Execute(ctx, "country", map[string]interface{}{
						"code": "ES",
					})
				} else {
					// List countries (no args needed)
					result, err = connector.Execute(ctx, "countries", nil)
				}

				if err != nil {
					fmt.Printf("   ❌ Error: %v\n", err)
				} else {
					fmt.Printf("   ✅ Result: %v\n", formatResult(result))
				}
				break
			}
		}
	}

	fmt.Println()
	fmt.Println("✨ Done!")
	fmt.Println()
	fmt.Println("These tools can be used with any Kairos agent:")
	fmt.Print(`
    agent := kairos.NewAgent(
        kairos.WithProvider(openaiProvider),
        kairos.WithTools(connector.Tools()...),
    )
`)
}

func formatResult(v interface{}) string {
	switch val := v.(type) {
	case map[string]interface{}:
		// Just show keys for brevity
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		return fmt.Sprintf("{keys: %v}", keys)
	default:
		s := fmt.Sprintf("%v", v)
		if len(s) > 100 {
			return s[:97] + "..."
		}
		return s
	}
}
