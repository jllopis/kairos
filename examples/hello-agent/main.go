package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/llm"
)

func main() {
	// 1. Define LLM (using Mock for hello-agent to be self-contained)
	provider := &llm.MockProvider{Response: "Hello from Kairos Agent!"}

	// 2. Create Agent
	a, err := agent.New("hello-agent", provider,
		agent.WithRole("Greeter"),
	)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// 3. Run Agent
	response, err := a.Run(context.Background(), "Say hello")
	if err != nil {
		log.Fatalf("agent run failed: %v", err)
	}

	fmt.Printf("Agent ID: %s\nRole: %s\nResponse: %v\n", a.ID(), a.Role(), response)
}
