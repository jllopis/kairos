package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/llm"
)

func main() {
	llmProvider := &llm.MockProvider{Response: "Final Answer: Extracted text from PDF successfully."}

	a, err := agent.New("skills-agent", llmProvider,
		agent.WithSkillsFromDir("./skills"),
	)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	fmt.Printf("Skills loaded: %d\n", len(a.Skills()))
	for _, skill := range a.Skills() {
		fmt.Printf("  - %s: %s\n", skill.Name, skill.Description)
	}

	resp, err := a.Run(context.Background(), "Extrae datos de este PDF")
	if err != nil {
		log.Fatalf("run agent: %v", err)
	}

	fmt.Printf("\nResponse: %v\n", resp)
}
