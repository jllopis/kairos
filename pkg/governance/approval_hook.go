// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package governance

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// StaticApprovalHook returns a fixed decision for every request.
type StaticApprovalHook struct {
	Decision Decision
}

// Request returns the configured decision.
func (h StaticApprovalHook) Request(_ context.Context, _ Action) Decision {
	return normalizeApprovalDecision(h.Decision, "approval decision not set")
}

// ConsoleApprovalHook prompts for approval on stdin/stdout.
type ConsoleApprovalHook struct {
	in      *bufio.Reader
	out     io.Writer
	prompt  string
	timeout time.Duration
	defaultDecision Decision
}

// ConsoleApprovalOption configures the console approval hook.
type ConsoleApprovalOption func(*ConsoleApprovalHook)

// NewConsoleApprovalHook creates a console-based approval hook.
func NewConsoleApprovalHook(opts ...ConsoleApprovalOption) *ConsoleApprovalHook {
	h := &ConsoleApprovalHook{
		in:     bufio.NewReader(os.Stdin),
		out:    os.Stdout,
		prompt: "Approve? [y/N]: ",
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// WithApprovalInput sets the input reader for the console hook.
func WithApprovalInput(r io.Reader) ConsoleApprovalOption {
	return func(h *ConsoleApprovalHook) {
		if r != nil {
			h.in = bufio.NewReader(r)
		}
	}
}

// WithApprovalOutput sets the output writer for the console hook.
func WithApprovalOutput(w io.Writer) ConsoleApprovalOption {
	return func(h *ConsoleApprovalHook) {
		if w != nil {
			h.out = w
		}
	}
}

// WithApprovalPrompt sets the prompt string.
func WithApprovalPrompt(prompt string) ConsoleApprovalOption {
	return func(h *ConsoleApprovalHook) {
		if strings.TrimSpace(prompt) != "" {
			h.prompt = prompt
		}
	}
}

// WithApprovalTimeout sets a timeout for waiting on user input.
func WithApprovalTimeout(timeout time.Duration) ConsoleApprovalOption {
	return func(h *ConsoleApprovalHook) {
		if timeout > 0 {
			h.timeout = timeout
		}
	}
}

// WithApprovalDefault sets the default decision when input is invalid or missing.
func WithApprovalDefault(decision Decision) ConsoleApprovalOption {
	return func(h *ConsoleApprovalHook) {
		h.defaultDecision = decision
	}
}

// Request prompts for approval and returns the operator decision.
func (h *ConsoleApprovalHook) Request(ctx context.Context, action Action) Decision {
	if h == nil || h.in == nil {
		return normalizeApprovalDecision(h.defaultDecision, "approval input not available")
	}

	reason := ""
	if action.Metadata != nil {
		reason = strings.TrimSpace(action.Metadata["policy_reason"])
	}
	if reason == "" {
		reason = "approval required"
	}

	_, _ = fmt.Fprintf(h.out, "\nApproval required for %s %q\n", action.Type, action.Name)
	if action.Metadata != nil {
		if ruleID := strings.TrimSpace(action.Metadata["policy_rule_id"]); ruleID != "" {
			_, _ = fmt.Fprintf(h.out, "Rule: %s\n", ruleID)
		}
	}
	_, _ = fmt.Fprintf(h.out, "Reason: %s\n", reason)
	_, _ = fmt.Fprint(h.out, h.prompt)

	responseCh := make(chan string, 1)
	go func() {
		line, _ := h.in.ReadString('\n')
		responseCh <- line
	}()

	if h.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.timeout)
		defer cancel()
	}

	select {
	case <-ctx.Done():
		return normalizeApprovalDecision(h.defaultDecision, "approval cancelled")
	case line := <-responseCh:
		answer := strings.ToLower(strings.TrimSpace(line))
		if strings.HasPrefix(answer, "y") {
			return Decision{Allowed: true, Status: DecisionStatusAllow, Reason: "approved by operator"}
		}
		return Decision{Allowed: false, Status: DecisionStatusDeny, Reason: "rejected by operator"}
	}
}

func normalizeApprovalDecision(decision Decision, fallbackReason string) Decision {
	if decision.Status == "" && decision.Reason == "" && !decision.Allowed {
		return Decision{Allowed: false, Status: DecisionStatusDeny, Reason: fallbackReason}
	}
	if decision.Status == "" {
		if decision.Allowed {
			decision.Status = DecisionStatusAllow
		} else {
			decision.Status = DecisionStatusDeny
		}
	}
	return decision
}
