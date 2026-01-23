package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/llm"
)

func main() {
	ctx := context.Background()
	provider := llm.NewScriptedMockProvider("mock", "You are SkyGuide, a travel assistant")

	agent, err := agent.New(
		"skyguide",
		provider,
	)
	if err != nil {
		slog.Error("failed to create agent", "error", err)
		return
	}

	response, err := agent.Run(ctx, "Hello")
	if err != nil {
		slog.Error("failed to run agent", "error", err)
		return
	}

	fmt.Println(response)
}
