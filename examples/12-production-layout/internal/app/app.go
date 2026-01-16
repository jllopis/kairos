// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

// Package app wires together all Kairos components.
// This is the "composition root" - all dependencies are created and connected here.
package app

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jllopis/kairos/examples/12-production-layout/internal/config"
	"github.com/jllopis/kairos/examples/12-production-layout/internal/observability"
	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/governance"
	"github.com/jllopis/kairos/pkg/llm"
)

// App holds the application state and components.
type App struct {
	cfg    *config.Config
	agent  *agent.Agent
	logger *slog.Logger
}

// New creates a new application with all components wired.
func New(cfg *config.Config) (*App, error) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.App.LogLevel),
	}))

	return &App{
		cfg:    cfg,
		logger: logger,
	}, nil
}

// Run starts the application main loop.
func (a *App) Run(ctx context.Context) error {
	// 1. Initialize observability first
	shutdown, err := observability.Init(ctx, a.cfg)
	if err != nil {
		return fmt.Errorf("init observability: %w", err)
	}
	defer shutdown(context.Background())

	a.logger.Info("starting application",
		"name", a.cfg.App.Name,
		"llm_provider", a.cfg.LLM.Provider,
	)

	// 2. Create LLM provider
	provider, err := a.createLLMProvider()
	if err != nil {
		return fmt.Errorf("create llm provider: %w", err)
	}

	// 3. Create governance engine (if enabled)
	var policyEngine governance.PolicyEngine
	if a.cfg.Governance.Enable {
		policyEngine = a.createGovernance()
	}

	// 4. Create agent with all options
	opts := []agent.Option{
		agent.WithModel(a.cfg.LLM.Model),
		agent.WithRole("Production Assistant"),
	}

	if policyEngine != nil {
		opts = append(opts, agent.WithPolicyEngine(policyEngine))
	}

	ag, err := agent.New(a.cfg.App.Name, provider, opts...)
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	a.agent = ag

	// 5. Run interactive loop
	return a.interactiveLoop(ctx)
}

func (a *App) createLLMProvider() (llm.Provider, error) {
	switch a.cfg.LLM.Provider {
	case "mock":
		return &llm.MockProvider{Response: "Mock response from Kairos agent"}, nil
	case "ollama":
		return llm.NewOllama(a.cfg.LLM.BaseURL), nil
	default:
		return nil, fmt.Errorf("unknown llm provider: %s", a.cfg.LLM.Provider)
	}
}

func (a *App) createGovernance() governance.PolicyEngine {
	rules := make([]governance.Rule, 0, len(a.cfg.Governance.Policies))
	for i, p := range a.cfg.Governance.Policies {
		rules = append(rules, governance.Rule{
			ID:     fmt.Sprintf("rule-%d", i),
			Effect: "allow",
			Name:   p,
		})
	}
	return governance.NewRuleSet(rules)
}

func (a *App) interactiveLoop(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("\n%s ready. Type your message (Ctrl+C to exit):\n\n", a.cfg.App.Name)

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		result, err := a.agent.Run(ctx, input)
		if err != nil {
			a.logger.Error("agent error", "error", err)
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		fmt.Printf("\n%s\n\n", result)
	}

	return scanner.Err()
}

func parseLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
