// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/governance"
	"github.com/jllopis/kairos/pkg/llm"
)

func main() {
	// Mock LLM - in production, skills would be invoked via the LLM's tool calling mechanism
	llmProvider := &llm.MockProvider{Response: "Final Answer: I've processed the PDF using the pdf-processing skill instructions."}

	// Optional: Configure governance tool filter for access control
	// This replaces the old skill-based tool filtering with centralized governance
	toolFilter := governance.NewToolFilter(
		governance.WithAllowlist([]string{"pdf-processing", "Bash(pdf:*)", "Bash(ocr:*)"}),
	)

	a, err := agent.New("skills-agent", llmProvider,
		agent.WithSkillsFromDir("./skills"),
		agent.WithToolFilter(toolFilter),
	)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	fmt.Println("=== Skills Agent Demo ===")
	fmt.Println("\nSkills are loaded as tools with progressive disclosure:")
	fmt.Println("  1. Metadata (name, description) → shown to LLM initially")
	fmt.Println("  2. Instructions (Body) → injected when LLM activates skill")
	fmt.Println("  3. Resources (scripts/, references/) → loaded on demand")

	fmt.Printf("\nSkills loaded: %d\n", len(a.Skills()))
	for _, skill := range a.Skills() {
		fmt.Printf("  - %s: %s\n", skill.Name, skill.Description)
	}

	fmt.Println("\n--- Running agent ---")
	resp, err := a.Run(context.Background(), "Extrae datos de este PDF")
	if err != nil {
		log.Fatalf("run agent: %v", err)
	}

	fmt.Printf("\nResponse: %v\n", resp)
}
