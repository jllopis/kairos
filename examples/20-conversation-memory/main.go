// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/pkg/memory"
)

func main() {
	// Get Ollama endpoint from environment or use default
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	// Get model from environment or use default
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.2" // Default model, adjust as needed
	}

	// Create Ollama LLM provider
	llmProvider := llm.NewOllama(ollamaURL)

	// Create conversation memory with window strategy
	// Keeps the last 20 messages, preserving system messages
	convMem := memory.NewInMemoryConversation(memory.ConversationConfig{
		TruncationStrategy: memory.NewWindowStrategy(20, true),
	})

	// Create agent with conversation memory
	a, err := agent.New("conversation-agent", llmProvider,
		agent.WithRole("Eres un asistente amigable que recuerda el contexto de la conversación."),
		agent.WithModel(model),
		agent.WithConversationMemory(convMem),
	)
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	fmt.Println("=== Conversation Memory Demo ===")
	fmt.Printf("Usando modelo: %s en %s\n", model, ollamaURL)
	fmt.Println("Este ejemplo demuestra cómo un agente mantiene el contexto")
	fmt.Println("entre múltiples interacciones usando ConversationMemory.")
	fmt.Println()

	// Create a session for this conversation
	sessionID := "demo-session-001"
	ctx := core.WithSessionID(context.Background(), sessionID)

	// Simulate a multi-turn conversation
	conversation := []string{
		"Hola, me llamo Juan",
		"¿Cómo me llamo?",
		"Quiero aprender a usar Kairos, ¿por dónde empiezo?",
		"¿De qué hemos hablado?",
	}

	for i, userMsg := range conversation {
		fmt.Printf("--- Turno %d ---\n", i+1)
		fmt.Printf("Usuario: %s\n", userMsg)

		resp, err := a.Run(ctx, userMsg)
		if err != nil {
			log.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("Agente: %v\n\n", resp)
	}

	// Show conversation history
	fmt.Println("=== Historial de la conversación ===")
	messages, _ := convMem.GetMessages(ctx, sessionID)
	for _, msg := range messages {
		role := msg.Role
		if role == "user" {
			role = "Usuario"
		} else if role == "assistant" {
			role = "Agente"
		}
		fmt.Printf("[%s] %s\n", role, truncate(msg.Content, 60))
	}

	fmt.Printf("\nTotal de mensajes almacenados: %d\n", len(messages))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
