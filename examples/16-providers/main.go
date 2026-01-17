// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Example 16: LLM Providers - Test authentication and token usage
//
// This example demonstrates how to use different LLM providers with Kairos.
// It tests authentication and shows token consumption for each provider.
//
// Usage:
//
//	# Test OpenAI (requires OPENAI_API_KEY env var)
//	go run . -provider openai
//
//	# Test Anthropic (requires ANTHROPIC_API_KEY env var)
//	go run . -provider anthropic
//
//	# Test Gemini (requires GOOGLE_API_KEY or GEMINI_API_KEY env var)
//	go run . -provider gemini
//
//	# Test Qwen (requires DASHSCOPE_API_KEY env var)
//	go run . -provider qwen
//
//	# Test all providers
//	go run . -provider all
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
	"github.com/jllopis/kairos/providers/qwen"
)

func main() {
	providerName := flag.String("provider", "openai", "Provider to test: openai, anthropic, gemini, qwen, all")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testPrompt := "Responde en una sola línea: ¿Cuál es la capital de España?"

	switch *providerName {
	case "openai":
		testOpenAI(ctx, testPrompt)
	case "anthropic":
		testAnthropic(ctx, testPrompt)
	case "gemini":
		testGemini(ctx, testPrompt)
	case "qwen":
		testQwen(ctx, testPrompt)
	case "all":
		testOpenAI(ctx, testPrompt)
		fmt.Println()
		testAnthropic(ctx, testPrompt)
		fmt.Println()
		testGemini(ctx, testPrompt)
		fmt.Println()
		testQwen(ctx, testPrompt)
	default:
		fmt.Printf("Unknown provider: %s\n", *providerName)
		os.Exit(1)
	}
}

func testOpenAI(ctx context.Context, prompt string) {
	fmt.Println("=== OpenAI Provider ===")

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ OPENAI_API_KEY not set")
		return
	}
	fmt.Println("✓ API key found")

	provider := openai.New() // Uses env var by default
	testProvider(ctx, provider, "gpt-5-mini", prompt)
}

func testAnthropic(ctx context.Context, prompt string) {
	fmt.Println("=== Anthropic Provider ===")

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ ANTHROPIC_API_KEY not set")
		return
	}
	fmt.Println("✓ API key found")

	provider := anthropic.New() // Uses env var by default
	testProvider(ctx, provider, "claude-haiku-4-20250514", prompt)
}

func testGemini(ctx context.Context, prompt string) {
	fmt.Println("=== Gemini Provider ===")

	apiKey := os.Getenv("GOOGLE_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	if apiKey == "" {
		fmt.Println("❌ GOOGLE_API_KEY or GEMINI_API_KEY not set")
		return
	}
	fmt.Println("✓ API key found")

	provider, err := gemini.New(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to create provider: %v\n", err)
		return
	}
	testProvider(ctx, provider, "gemini-3-flash-preview", prompt)
}

func testQwen(ctx context.Context, prompt string) {
	fmt.Println("=== Qwen Provider ===")

	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ DASHSCOPE_API_KEY not set")
		return
	}
	fmt.Println("✓ API key found")

	provider := qwen.New(apiKey)
	testProvider(ctx, provider, "qwen-turbo", prompt)
}

func testProvider(ctx context.Context, provider llm.Provider, model, prompt string) {
	fmt.Printf("Model: %s\n", model)
	fmt.Printf("Prompt: %s\n", prompt)

	start := time.Now()

	req := llm.ChatRequest{
		Model: model,
		Messages: []llm.Message{
			{Role: llm.RoleUser, Content: prompt},
		},
	}

	resp, err := provider.Chat(ctx, req)
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		return
	}

	elapsed := time.Since(start)

	fmt.Println("---")
	fmt.Printf("✓ Response: %s\n", resp.Content)
	fmt.Println("---")
	fmt.Printf("Token Usage:\n")
	fmt.Printf("  Prompt tokens:     %d\n", resp.Usage.PromptTokens)
	fmt.Printf("  Completion tokens: %d\n", resp.Usage.CompletionTokens)
	fmt.Printf("  Total tokens:      %d\n", resp.Usage.TotalTokens)
	fmt.Printf("  Latency:           %v\n", elapsed)
}
