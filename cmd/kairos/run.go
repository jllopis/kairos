// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/governance"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/pkg/planner"
	"github.com/jllopis/kairos/pkg/telemetry"
	"github.com/mattn/go-isatty"
)

func runRun(ctx context.Context, flags globalFlags, args []string) {
	cmd := flag.NewFlagSet("run", flag.ContinueOnError)
	profile := cmd.String("profile", "", "Config profile to load (dev, prod)")
	agentID := cmd.String("agent", "kairos-agent", "Agent ID")
	role := cmd.String("role", "Helpful Assistant", "Agent role")
	prompt := cmd.String("prompt", "", "Single prompt to run (non-interactive)")
	skillsDir := cmd.String("skills", "", "Directory containing skills")
	planPath := cmd.String("plan", "", "Path to explicit planner graph (YAML/JSON)")
	interactive := cmd.Bool("interactive", true, "Run in interactive REPL mode")
	noTelemetry := cmd.Bool("no-telemetry", false, "Disable telemetry output")
	watch := cmd.Bool("watch", false, "Watch config files for changes and hot-reload")
	approvalMode := cmd.String("approval-mode", "auto", "Local approvals: auto|ask|approve|deny|off")
	approvalTimeout := cmd.Duration("approval-timeout", 0, "Timeout for local approval prompt")

	if err := cmd.Parse(args); err != nil {
		fatal(err)
	}

	// Load config with profile
	configArgs := flags.ConfigArgs
	if *profile != "" {
		// Add profile-specific config
		profilePath := fmt.Sprintf("./config/config.%s.yaml", *profile)
		if _, err := os.Stat(profilePath); err == nil {
			configArgs = append(configArgs, "--config", profilePath)
		}
	}

	cfg, err := config.LoadWithCLI(configArgs)
	if err != nil {
		fatal(fmt.Errorf("failed to load config: %w", err))
	}

	// Setup config watcher if requested
	var configWatcher *config.Watcher
	reloadableCfg := config.NewReloadableConfig(cfg)
	
	if *watch {
		// Find config file path for watching
		configPath := findConfigPath(configArgs)
		if configPath != "" {
			var watchErr error
			configWatcher, _, watchErr = config.WatchConfig(ctx, configPath,
				config.WithWatchInterval(1*time.Second),
			)
			if watchErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not setup config watch: %v\n", watchErr)
			} else {
				configWatcher.OnChange(func(newCfg *config.Config) {
					reloadableCfg.Update(newCfg)
					if !flags.JSON {
						fmt.Println("\n[Config reloaded]")
					}
				})
				if !flags.JSON {
					fmt.Printf("Watching config: %s\n", configPath)
				}
			}
		}
	}
	defer func() {
		if configWatcher != nil {
			configWatcher.Stop()
		}
	}()

	// Initialize telemetry (use "none" if disabled or for cleaner CLI output)
	exporter := cfg.Telemetry.Exporter
	if *noTelemetry || exporter == "" {
		exporter = "none"
	}
	shutdown, err := telemetry.InitWithConfig(*agentID, "v0.1.0", telemetry.Config{
		Exporter:           exporter,
		OTLPEndpoint:       cfg.Telemetry.OTLPEndpoint,
		OTLPInsecure:       cfg.Telemetry.OTLPInsecure,
		OTLPTimeoutSeconds: cfg.Telemetry.OTLPTimeoutSeconds,
	})
	if err != nil {
		fatal(fmt.Errorf("failed to init telemetry: %w", err))
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "telemetry shutdown: %v\n", err)
		}
	}()

	// Setup LLM provider
	provider, err := createProvider(cfg)
	if err != nil {
		fatal(err)
	}

	// Build agent options
	opts := []agent.Option{
		agent.WithRole(*role),
		agent.WithModel(cfg.LLM.Model),
	}

	// Add governance if configured
	if len(cfg.Governance.Policies) > 0 {
		policy := governance.RuleSetFromConfig(cfg.Governance)
		opts = append(opts, agent.WithPolicyEngine(policy))
		timeout := *approvalTimeout
		if timeout == 0 && cfg.Governance.ApprovalTimeoutSeconds > 0 {
			timeout = time.Duration(cfg.Governance.ApprovalTimeoutSeconds) * time.Second
		}
		hook := buildApprovalHook(*approvalMode, timeout, cfg.Governance.Policies, flags.JSON)
		if hook != nil {
			opts = append(opts, agent.WithApprovalHook(hook))
		}
	}

	// Add skills if directory specified
	if *skillsDir != "" {
		opts = append(opts, agent.WithSkillsFromDir(*skillsDir))
	}

	// Add explicit planner if specified
	if strings.TrimSpace(*planPath) != "" {
		graph, err := planner.LoadGraph(*planPath)
		if err != nil {
			fatal(fmt.Errorf("failed to load plan: %w", err))
		}
		opts = append(opts, agent.WithPlanner(graph))
	}

	// Add MCP servers if configured
	if len(cfg.MCP.Servers) > 0 {
		opts = append(opts, agent.WithMCPServerConfigs(cfg.MCP.Servers))
	}

	// Agent-specific config
	agentCfg := cfg.AgentConfigFor(*agentID)
	opts = append(opts,
		agent.WithDisableActionFallback(agentCfg.DisableActionFallback),
		agent.WithActionFallbackWarning(agentCfg.WarnOnActionFallback),
	)

	// Create agent
	ag, err := agent.New(*agentID, provider, opts...)
	if err != nil {
		fatal(fmt.Errorf("failed to create agent: %w", err))
	}
	defer func() {
		if err := ag.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "agent close: %v\n", err)
		}
	}()

	// Print startup info
	if !flags.JSON {
		fmt.Printf("Kairos Agent: %s\n", *agentID)
		fmt.Printf("LLM: %s (%s)\n", cfg.LLM.Provider, cfg.LLM.Model)
		if strings.TrimSpace(*planPath) != "" {
			fmt.Printf("Planner: %s\n", *planPath)
		}
		if len(cfg.MCP.Servers) > 0 {
			fmt.Printf("MCP Servers: %d\n", len(cfg.MCP.Servers))
		}
		if len(cfg.Governance.Policies) > 0 {
			fmt.Printf("Policies: %d\n", len(cfg.Governance.Policies))
		}
		fmt.Println()
	}

	// Handle graceful shutdown
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Single prompt mode
	if *prompt != "" {
		runSinglePrompt(ctx, ag, *prompt, flags.JSON)
		return
	}

	// Interactive REPL mode
	if *interactive {
		runREPL(ctx, ag, flags.JSON)
		return
	}

	// Read from stdin (pipe mode)
	runPipeMode(ctx, ag, flags.JSON)
}

func createProvider(cfg *config.Config) (llm.Provider, error) {
	switch strings.ToLower(cfg.LLM.Provider) {
	case "ollama":
		baseURL := cfg.LLM.BaseURL
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
		return llm.NewOllama(baseURL), nil

	case "mock":
		return &llm.MockProvider{Response: "This is a mock response."}, nil

	case "":
		// Default to ollama
		return llm.NewOllama("http://localhost:11434"), nil

	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.LLM.Provider)
	}
}

func runSinglePrompt(ctx context.Context, ag *agent.Agent, prompt string, jsonOutput bool) {
	response, err := ag.Run(ctx, prompt)
	if err != nil {
		if jsonOutput {
			printJSON(map[string]string{"error": err.Error()})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}

	if jsonOutput {
		printJSON(map[string]string{
			"prompt":   prompt,
			"response": fmt.Sprintf("%v", response),
		})
	} else {
		fmt.Printf("%v\n", response)
	}
}

func runREPL(ctx context.Context, ag *agent.Agent, jsonOutput bool) {
	if !jsonOutput {
		fmt.Println("Interactive mode. Type 'exit' or Ctrl+C to quit.")
		fmt.Println("---")
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		if !jsonOutput {
			fmt.Print("\n> ")
		}

		select {
		case <-ctx.Done():
			if !jsonOutput {
				fmt.Println("\nGoodbye!")
			}
			return
		default:
		}

		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			if !jsonOutput {
				fmt.Println("Goodbye!")
			}
			return
		}

		// Special commands
		if strings.HasPrefix(input, "/") {
			handleCommand(ag, input, jsonOutput)
			continue
		}

		response, err := ag.Run(ctx, input)
		if err != nil {
			if jsonOutput {
				printJSON(map[string]string{"error": err.Error()})
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			continue
		}

		if jsonOutput {
			printJSON(map[string]string{
				"prompt":   input,
				"response": fmt.Sprintf("%v", response),
			})
		} else {
			fmt.Printf("\n%v\n", response)
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Read error: %v\n", err)
	}
}

func runPipeMode(ctx context.Context, ag *agent.Agent, jsonOutput bool) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		response, err := ag.Run(ctx, input)
		if err != nil {
			if jsonOutput {
				printJSON(map[string]string{"error": err.Error()})
			} else {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			}
			continue
		}

		if jsonOutput {
			printJSON(map[string]string{
				"prompt":   input,
				"response": fmt.Sprintf("%v", response),
			})
		} else {
			fmt.Printf("%v\n", response)
		}
	}
}

func handleCommand(ag *agent.Agent, input string, jsonOutput bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "/help":
		if !jsonOutput {
			fmt.Println(`Commands:
  /help     Show this help
  /tools    List available tools
  /skills   List loaded skills
  /clear    Clear conversation (not implemented)
  /exit     Exit the REPL`)
		}

	case "/tools":
		tools := ag.ToolNames()
		if jsonOutput {
			printJSON(map[string]interface{}{"tools": tools})
		} else {
			if len(tools) == 0 {
				fmt.Println("No tools available")
			} else {
				fmt.Println("Available tools:")
				for _, t := range tools {
					fmt.Printf("  - %s\n", t)
				}
			}
		}

	case "/skills":
		skills := ag.Skills()
		if jsonOutput {
			names := make([]string, len(skills))
			for i, s := range skills {
				names[i] = s.Name
			}
			printJSON(map[string]interface{}{"skills": names})
		} else {
			if len(skills) == 0 {
				fmt.Println("No skills loaded")
			} else {
				fmt.Println("Loaded skills:")
				for _, s := range skills {
					fmt.Printf("  - %s: %s\n", s.Name, truncateString(s.Description, 50))
				}
			}
		}

	case "/exit", "/quit":
		if !jsonOutput {
			fmt.Println("Goodbye!")
		}
		os.Exit(0)

	default:
		if !jsonOutput {
			fmt.Printf("Unknown command: %s (try /help)\n", cmd)
		}
	}
}

// findConfigPath extracts the config path from CLI args.
func findConfigPath(args []string) string {
	for i, arg := range args {
		if arg == "--config" && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config=")
		}
	}
	// Check default locations
	for _, path := range []string{
		"./.kairos/config.yaml",
		"./.kairos/settings.yaml",
		"./config.yaml",
	} {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func buildApprovalHook(mode string, timeout time.Duration, policies []config.PolicyRuleConfig, jsonOutput bool) governance.ApprovalHook {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "auto"
	}
	if mode == "off" || mode == "disabled" || mode == "none" {
		return nil
	}
	if mode == "auto" && !hasPendingPolicies(policies) {
		return nil
	}

	isTTY := isatty.IsTerminal(os.Stdin.Fd()) && isatty.IsTerminal(os.Stdout.Fd())
	if jsonOutput {
		isTTY = false
	}

	if mode == "auto" {
		if isTTY {
			mode = "ask"
		} else {
			mode = "deny"
		}
	}
	if mode == "ask" && !isTTY {
		fmt.Fprintln(os.Stderr, "Approval mode 'ask' requires a TTY; falling back to deny.")
		mode = "deny"
	}

	switch mode {
	case "ask":
		opts := []governance.ConsoleApprovalOption{}
		if timeout > 0 {
			opts = append(opts, governance.WithApprovalTimeout(timeout))
		}
		return governance.NewConsoleApprovalHook(opts...)
	case "approve":
		return governance.StaticApprovalHook{
			Decision: governance.Decision{
				Allowed: true,
				Status:  governance.DecisionStatusAllow,
				Reason:  "auto-approved",
			},
		}
	case "deny":
		return governance.StaticApprovalHook{
			Decision: governance.Decision{
				Allowed: false,
				Status:  governance.DecisionStatusDeny,
				Reason:  "auto-denied",
			},
		}
	default:
		return nil
	}
}

func hasPendingPolicies(policies []config.PolicyRuleConfig) bool {
	for _, rule := range policies {
		if strings.ToLower(strings.TrimSpace(rule.Effect)) == "pending" {
			return true
		}
	}
	return false
}
