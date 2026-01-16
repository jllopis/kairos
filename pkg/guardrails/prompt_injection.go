// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"context"
	"regexp"
	"strings"
)

// PromptInjectionDetector detects potential prompt injection attacks.
// It uses pattern matching to identify common injection techniques.
type PromptInjectionDetector struct {
	patterns   []*regexp.Regexp
	threshold  float64
	strictMode bool
}

// PromptInjectionOption configures the prompt injection detector.
type PromptInjectionOption func(*PromptInjectionDetector)

// Common prompt injection patterns
var defaultInjectionPatterns = []string{
	// Direct instruction override attempts
	`(?i)ignore\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|rules?)`,
	`(?i)disregard\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|rules?)`,
	`(?i)forget\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|rules?)`,
	`(?i)override\s+(all\s+)?(previous|prior|above)\s+(instructions?|prompts?|rules?)`,

	// Role/persona manipulation
	`(?i)you\s+are\s+now\s+(a|an)\s+`,
	`(?i)pretend\s+(you\s+are|to\s+be)\s+`,
	`(?i)act\s+as\s+(a|an|if)\s+`,
	`(?i)roleplay\s+as\s+`,
	`(?i)switch\s+to\s+.*\s+mode`,
	`(?i)enter\s+.*\s+mode`,

	// System prompt extraction
	`(?i)what\s+(is|are)\s+your\s+(system\s+)?(prompt|instructions?)`,
	`(?i)show\s+me\s+your\s+(system\s+)?(prompt|instructions?)`,
	`(?i)reveal\s+your\s+(system\s+)?(prompt|instructions?)`,
	`(?i)print\s+your\s+(system\s+)?(prompt|instructions?)`,
	`(?i)display\s+your\s+(system\s+)?(prompt|instructions?)`,

	// Jailbreak attempts
	`(?i)do\s+anything\s+now`,
	`(?i)DAN\s+mode`,
	`(?i)jailbreak`,
	`(?i)bypass\s+(safety|content|filter)`,

	// Developer/debug mode attempts
	`(?i)developer\s+mode`,
	`(?i)debug\s+mode`,
	`(?i)sudo\s+mode`,
	`(?i)admin\s+mode`,
	`(?i)maintenance\s+mode`,

	// Encoding/obfuscation indicators
	`(?i)base64\s+(decode|encode)`,
	`(?i)rot13`,
	`(?i)execute\s+(this\s+)?(code|command)`,

	// Delimiter manipulation
	`(?i)\]\]\s*system\s*:`,
	`(?i)<\|.*\|>`,
	`(?i)\[INST\]`,
	`(?i)\[\/INST\]`,
	`(?i)<<SYS>>`,
	`(?i)<</SYS>>`,
}

// NewPromptInjectionDetector creates a new prompt injection detector.
func NewPromptInjectionDetector(opts ...PromptInjectionOption) *PromptInjectionDetector {
	d := &PromptInjectionDetector{
		patterns:   make([]*regexp.Regexp, 0, len(defaultInjectionPatterns)),
		threshold:  0.0, // Block on any match by default (secure)
		strictMode: false,
	}

	// Compile default patterns
	for _, pattern := range defaultInjectionPatterns {
		if re, err := regexp.Compile(pattern); err == nil {
			d.patterns = append(d.patterns, re)
		}
	}

	for _, opt := range opts {
		opt(d)
	}

	return d
}

// WithInjectionPatterns adds custom patterns to detect.
func WithInjectionPatterns(patterns []string) PromptInjectionOption {
	return func(d *PromptInjectionDetector) {
		for _, pattern := range patterns {
			if re, err := regexp.Compile(pattern); err == nil {
				d.patterns = append(d.patterns, re)
			}
		}
	}
}

// WithInjectionThreshold sets the confidence threshold for detection.
func WithInjectionThreshold(threshold float64) PromptInjectionOption {
	return func(d *PromptInjectionDetector) {
		if threshold >= 0 && threshold <= 1 {
			d.threshold = threshold
		}
	}
}

// WithStrictMode enables strict mode which blocks on any pattern match.
func WithStrictMode(strict bool) PromptInjectionOption {
	return func(d *PromptInjectionDetector) {
		d.strictMode = strict
	}
}

// ID returns the guardrail identifier.
func (d *PromptInjectionDetector) ID() string {
	return "prompt-injection"
}

// CheckInput analyzes input for prompt injection attempts.
func (d *PromptInjectionDetector) CheckInput(ctx context.Context, input string) CheckResult {
	if input == "" {
		return CheckResult{Blocked: false}
	}

	// Normalize input for detection
	normalized := strings.ToLower(input)
	matchCount := 0
	var matchedPatterns []string

	for _, pattern := range d.patterns {
		select {
		case <-ctx.Done():
			return CheckResult{Blocked: false}
		default:
		}

		if pattern.MatchString(normalized) {
			matchCount++
			matchedPatterns = append(matchedPatterns, pattern.String())

			// In strict mode, block on first match
			if d.strictMode {
				return CheckResult{
					Blocked:     true,
					Reason:      "potential prompt injection detected",
					GuardrailID: d.ID(),
					Confidence:  1.0,
					Metadata: map[string]any{
						"matched_patterns": matchedPatterns,
					},
				}
			}
		}
	}

	// Calculate confidence based on number of pattern matches
	if matchCount > 0 {
		// Base confidence starts at 0.7 for single match
		confidence := 0.7 + (float64(matchCount-1) * 0.1)
		if confidence > 1.0 {
			confidence = 1.0
		}

		if confidence >= d.threshold {
			return CheckResult{
				Blocked:     true,
				Reason:      "potential prompt injection detected",
				GuardrailID: d.ID(),
				Confidence:  confidence,
				Metadata: map[string]any{
					"matched_patterns": matchedPatterns,
					"match_count":      matchCount,
				},
			}
		}
	}

	return CheckResult{
		Blocked:    false,
		Confidence: 0,
	}
}

// WithPromptInjectionDetector returns an option that adds prompt injection detection.
func WithPromptInjectionDetector(opts ...PromptInjectionOption) Option {
	return func(g *Guardrails) {
		detector := NewPromptInjectionDetector(opts...)
		g.inputCheckers = append(g.inputCheckers, detector)
	}
}
