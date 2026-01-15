package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/llm"
)

func main() {
	llmProvider := &llm.MockProvider{Response: "Final Answer: listo"}

	a, err := agent.New("skills-agent", llmProvider,
		agent.WithSkillsFromDir("./examples/skills-agent/skills"),
	)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	resp, err := a.Run(context.Background(), "Extrae datos de este PDF")
	if err != nil {
		log.Fatalf("run agent: %v", err)
	}

	fmt.Printf("Skills loaded: %d\n", len(a.Skills()))
	fmt.Printf("Response: %v\n", resp)
}
