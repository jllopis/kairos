package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/memory/ollama"
	"github.com/jllopis/kairos/pkg/memory/qdrant"
	"github.com/jllopis/kairos/pkg/telemetry"
)

func main() {
	ctx := context.Background()

	// 1. Load Configuration
	cfg, err := config.Load("")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	// Force enable memory for this example
	cfg.Memory.Enabled = true

	// 2. Initialize Telemetry
	shutdown, err := telemetry.InitWithConfig("memory-agent", "v0.1.0", telemetry.Config{
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

	fmt.Printf("Starting Memory Agent...\n")

	// 3. Setup LLM Provider
	var provider llm.Provider
	switch cfg.LLM.Provider {
	case "ollama":
		provider = llm.NewOllama(cfg.LLM.BaseURL)
	default:
		provider = &llm.MockProvider{Response: "I am a mocked response."}
	}

	// 4. Setup Memory
	var mem core.Memory
	if cfg.Memory.Enabled {
		fmt.Printf("Initializing Memory (Qdrant: %s, Embedder: %s)...\n", cfg.Memory.QdrantAddr, cfg.Memory.EmbedderModel)

		qStore, err := qdrant.New(cfg.Memory.QdrantAddr)
		if err != nil {
			log.Printf("WARNING: Failed to connect to Qdrant: %v. Memory will be disabled.", err)
		} else {
			embedder := ollama.NewEmbedder(cfg.Memory.EmbedderBaseURL, cfg.Memory.EmbedderModel)

			vMem, err := memory.NewVectorMemory(ctx, qStore, embedder, "kairos_memory")
			if err != nil {
				log.Fatalf("failed to create vector memory: %v", err)
			}

			if err := vMem.Initialize(ctx); err != nil {
				log.Printf("WARNING: Failed to initialize vector memory (is Qdrant running?): %v. Memory might not work.", err)
			} else {
				mem = vMem
				fmt.Println("Memory initialized successfully.")
			}
		}
	}

	// 5. Create Agent
	a, err := agent.New("memory-assistant", provider,
		agent.WithRole("Helpful Assistant with Memory"),
		agent.WithModel(cfg.LLM.Model),
		agent.WithMemory(mem),
	)
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	// 6. Run Interactions

	// Interaction 1: Teach something
	input1 := "My favorite color is blue."
	fmt.Printf("\nUSER: %s\n", input1)
	resp1, err := a.Run(ctx, input1)
	if err != nil {
		log.Fatalf("run 1 failed: %v", err)
	}
	fmt.Printf("AGENT: %s\n", resp1)

	// Wait a bit for async indexing if any (Qdrant is usually fast but good measure)
	time.Sleep(1 * time.Second)

	// Interaction 2: Ask about it
	input2 := "What is my favorite color?"
	fmt.Printf("\nUSER: %s\n", input2)
	resp2, err := a.Run(ctx, input2)
	if err != nil {
		log.Fatalf("run 2 failed: %v", err)
	}
	fmt.Printf("AGENT: %s\n", resp2)
}
