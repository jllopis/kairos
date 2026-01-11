package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/pkg/telemetry"
)

func main() {
	ctx := context.Background()

	// 1. Load Configuration
	// Example: KAIROS_AGENT_DISABLE_ACTION_FALLBACK=true
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Initialize Telemetry
	shutdown, err := telemetry.InitWithConfig("basic-agent", "v0.2.5", telemetry.Config{
		Exporter:           cfg.Telemetry.Exporter,
		OTLPEndpoint:       cfg.Telemetry.OTLPEndpoint,
		OTLPInsecure:       cfg.Telemetry.OTLPInsecure,
		OTLPTimeoutSeconds: cfg.Telemetry.OTLPTimeoutSeconds,
	})
	if err != nil {
		log.Fatalf("failed to init telemetry: %v", err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
			log.Printf("telemetry shutdown failed: %v", err)
		}
	}()

	fmt.Printf("Starting Agent with Provider: %s, Model: %s\n", cfg.LLM.Provider, cfg.LLM.Model)

	// 3. Setup LLM Provider
	var provider llm.Provider
	switch cfg.LLM.Provider {
	case "ollama":
		provider = llm.NewOllama(cfg.LLM.BaseURL)
	default:
		// Fallback to Mock if configured or unknown
		provider = &llm.MockProvider{Response: "I am a mocked response."}
	}

	// 4. Create Agent
	agentCfg := cfg.AgentConfigFor("basic-assistant")
	a, err := agent.New("basic-assistant", provider,
		agent.WithRole("Helpful Assistant"),
		agent.WithModel(cfg.LLM.Model),
		agent.WithDisableActionFallback(agentCfg.DisableActionFallback),
	)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// 5. Run Interaction
	input := "Explain the concept of Agentic AI briefly."
	fmt.Printf("\nUSER: %s\n", input)

	response, err := a.Run(ctx, input)
	if err != nil {
		log.Fatalf("run failed: %v", err)
	}

	fmt.Printf("\nAGENT: %s\n", response)
}
