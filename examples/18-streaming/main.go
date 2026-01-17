// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Example 18: Streaming - Real-time LLM responses
//
// This example demonstrates how to use streaming with LLM providers
// to receive responses in real-time, token by token.
//
// Usage:
//
//	# Test with OpenAI (requires OPENAI_API_KEY)
//	go run . -provider openai
//
//	# Test with Anthropic (requires ANTHROPIC_API_KEY)
//	go run . -provider anthropic
//
//	# Test with Gemini (requires GOOGLE_API_KEY)
//	go run . -provider gemini
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/providers/anthropic"
	"github.com/jllopis/kairos/providers/gemini"
	"github.com/jllopis/kairos/providers/openai"
	"github.com/jllopis/kairos/pkg/llm"
)

func main() {
	providerName := flag.String("provider", "openai", "Provider to test: openai, anthropic, ollama, gemini")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	prompt := "Escribe un poema corto (4 versos) sobre la programación en Go."

	fmt.Printf("Provider: %s\n", *providerName)
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Println("---")
	fmt.Println("Streaming response:")
	fmt.Println()

	var provider llm.StreamingProvider
	var err error

	switch *providerName {
	case "openai":
		if os.Getenv("OPENAI_API_KEY") == "" {
			fmt.Println("❌ OPENAI_API_KEY not set")
			return
		}
		provider = openai.New()

	case "anthropic":
		if os.Getenv("ANTHROPIC_API_KEY") == "" {
			fmt.Println("❌ ANTHROPIC_API_KEY not set")
			return
		}
		provider = anthropic.New()

	case "ollama":
		provider = .New()

	case "gemini":
		apiKey := os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			apiKey = os.Getenv("GEMINI_API_KEY")
		}
		if apiKey == "" {
			fmt.Println("❌ GOOGLE_API_KEY or GEMINI_API_KEY not set")
			return
		}
		provider, err = gemini.New(ctx)
		if err != nil {
			fmt.Printf("❌ Failed to create Gemini provider: %v\n", err)
			return
		}

	default:
		fmt.Printf("Unknown provider: %s\n", *providerName)
		return
	}

	// Create chat request
	req := llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompt},
		},
	}

	// Start streaming
	start := time.Now()
	chunks, err := provider.ChatStream(ctx, req)
	if err != nil {
		fmt.Printf("❌ Failed to start stream: %v\n", err)
		return
	}

	// Process chunks
	var totalContent string
	var usage *llm.Usage
	firstChunkTime := time.Duration(0)
	chunkCount := 0

	for chunk := range chunks {
		if chunk.Error != nil {
			fmt.Printf("\n❌ Stream error: %v\n", chunk.Error)
			return
		}

		if chunkCount == 0 && chunk.Content != "" {
			firstChunkTime = time.Since(start)
		}

		// Print content as it arrives
		if chunk.Content != "" {
			fmt.Print(chunk.Content)
			totalContent += chunk.Content
			chunkCount++
		}

		// Capture usage from final chunk
		if chunk.Usage != nil {
			usage = chunk.Usage
		}

		if chunk.Done {
			break
		}
	}

	elapsed := time.Since(start)

	fmt.Println()
	fmt.Println()
	fmt.Println("---")
	fmt.Printf("✓ Streaming completed\n")
	fmt.Printf("  Total chunks:        %d\n", chunkCount)
	fmt.Printf("  Time to first chunk: %v\n", firstChunkTime)
	fmt.Printf("  Total time:          %v\n", elapsed)
	fmt.Printf("  Content length:      %d chars\n", len(totalContent))

	if usage != nil {
		fmt.Printf("  Token usage:\n")
		fmt.Printf("    Prompt:     %d\n", usage.PromptTokens)
		fmt.Printf("    Completion: %d\n", usage.CompletionTokens)
		fmt.Printf("    Total:      %d\n", usage.TotalTokens)
	}
}
