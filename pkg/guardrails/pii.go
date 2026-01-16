// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"context"
	"regexp"
	"strings"
)

// PIIFilterMode determines how PII is handled.
type PIIFilterMode int

const (
	// PIIFilterMask replaces PII with masked placeholders (e.g., "[EMAIL]").
	PIIFilterMask PIIFilterMode = iota
	// PIIFilterRedact removes PII entirely.
	PIIFilterRedact
	// PIIFilterHash replaces PII with a hash (for correlation without exposure).
	PIIFilterHash
)

// PIIType categorizes different types of PII.
type PIIType string

const (
	PIITypeEmail       PIIType = "email"
	PIITypePhone       PIIType = "phone"
	PIITypeSSN         PIIType = "ssn"
	PIITypeCreditCard  PIIType = "credit_card"
	PIITypeIPAddress   PIIType = "ip_address"
	PIITypeName        PIIType = "name"
	PIITypeAddress     PIIType = "address"
	PIITypePassport    PIIType = "passport"
	PIITypeDateOfBirth PIIType = "date_of_birth"
)

// piiPattern defines a pattern for detecting a type of PII.
type piiPattern struct {
	piiType PIIType
	pattern *regexp.Regexp
	mask    string
}

// PIIFilter detects and filters Personally Identifiable Information.
type PIIFilter struct {
	mode       PIIFilterMode
	patterns   []piiPattern
	enabledPII map[PIIType]bool
}

// PIIFilterOption configures the PII filter.
type PIIFilterOption func(*PIIFilter)

// Default PII patterns (conservative, high-precision)
// Order matters - more specific patterns should come first
var defaultPIIPatterns = []struct {
	piiType PIIType
	pattern string
	mask    string
}{
	// Credit card numbers (check before phone due to overlap)
	{PIITypeCreditCard, `\b[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}[-\s]?[0-9]{4}\b`, "[CREDIT_CARD]"},
	{PIITypeCreditCard, `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b`, "[CREDIT_CARD]"},

	// Social Security Numbers (US) - check before phone
	{PIITypeSSN, `\b[0-9]{3}[-\s]?[0-9]{2}[-\s]?[0-9]{4}\b`, "[SSN]"},

	// Email addresses
	{PIITypeEmail, `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`, "[EMAIL]"},

	// Phone numbers (various formats)
	{PIITypePhone, `\+?1?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}`, "[PHONE]"},
	{PIITypePhone, `\+[0-9]{1,3}[-.\s]?[0-9]{6,14}`, "[PHONE]"},

	// IP addresses (IPv4)
	{PIITypeIPAddress, `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`, "[IP_ADDRESS]"},

	// Dates that could be DOB (various formats)
	{PIITypeDateOfBirth, `\b(?:0?[1-9]|1[0-2])[/-](?:0?[1-9]|[12][0-9]|3[01])[/-](?:19|20)[0-9]{2}\b`, "[DATE]"},
	{PIITypeDateOfBirth, `\b(?:19|20)[0-9]{2}[/-](?:0?[1-9]|1[0-2])[/-](?:0?[1-9]|[12][0-9]|3[01])\b`, "[DATE]"},

	// Passport numbers (generic pattern)
	{PIITypePassport, `\b[A-Z]{1,2}[0-9]{6,9}\b`, "[PASSPORT]"},
}

// NewPIIFilter creates a new PII filter.
func NewPIIFilter(mode PIIFilterMode, opts ...PIIFilterOption) *PIIFilter {
	f := &PIIFilter{
		mode:       mode,
		patterns:   make([]piiPattern, 0),
		enabledPII: make(map[PIIType]bool),
	}

	// Enable all PII types by default
	for _, p := range defaultPIIPatterns {
		f.enabledPII[p.piiType] = true
	}

	// Compile default patterns
	for _, p := range defaultPIIPatterns {
		if re, err := regexp.Compile(p.pattern); err == nil {
			f.patterns = append(f.patterns, piiPattern{
				piiType: p.piiType,
				pattern: re,
				mask:    p.mask,
			})
		}
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}

// WithPIITypes enables only specific PII types.
func WithPIITypes(types ...PIIType) PIIFilterOption {
	return func(f *PIIFilter) {
		// Disable all first
		for k := range f.enabledPII {
			f.enabledPII[k] = false
		}
		// Enable specified
		for _, t := range types {
			f.enabledPII[t] = true
		}
	}
}

// WithExcludePII excludes specific PII types from filtering.
func WithExcludePII(types ...PIIType) PIIFilterOption {
	return func(f *PIIFilter) {
		for _, t := range types {
			f.enabledPII[t] = false
		}
	}
}

// WithCustomPIIPattern adds a custom PII pattern.
func WithCustomPIIPattern(piiType PIIType, pattern, mask string) PIIFilterOption {
	return func(f *PIIFilter) {
		if re, err := regexp.Compile(pattern); err == nil {
			f.patterns = append(f.patterns, piiPattern{
				piiType: piiType,
				pattern: re,
				mask:    mask,
			})
			f.enabledPII[piiType] = true
		}
	}
}

// ID returns the guardrail identifier.
func (f *PIIFilter) ID() string {
	return "pii-filter"
}

// FilterOutput processes output and masks/removes PII.
func (f *PIIFilter) FilterOutput(ctx context.Context, output string) FilterResult {
	if output == "" {
		return FilterResult{Content: output, Modified: false}
	}

	result := FilterResult{
		Content:    output,
		Modified:   false,
		Redactions: make([]Redaction, 0),
	}

	// Process each enabled pattern
	for _, p := range f.patterns {
		if !f.enabledPII[p.piiType] {
			continue
		}

		select {
		case <-ctx.Done():
			return result
		default:
		}

		matches := p.pattern.FindAllStringIndex(result.Content, -1)
		if len(matches) == 0 {
			continue
		}

		// Process matches in reverse order to preserve positions
		for i := len(matches) - 1; i >= 0; i-- {
			match := matches[i]
			original := result.Content[match[0]:match[1]]
			replacement := f.getReplacement(p, original)

			result.Redactions = append(result.Redactions, Redaction{
				Type:        string(p.piiType),
				Original:    "", // Don't expose PII in redaction log
				Replacement: replacement,
				Position:    match[0],
			})

			result.Content = result.Content[:match[0]] + replacement + result.Content[match[1]:]
			result.Modified = true
		}
	}

	return result
}

func (f *PIIFilter) getReplacement(p piiPattern, original string) string {
	switch f.mode {
	case PIIFilterMask:
		return p.mask
	case PIIFilterRedact:
		return ""
	case PIIFilterHash:
		// Simple hash representation (not cryptographic)
		return p.mask[:len(p.mask)-1] + "_" + hashString(original)[:8] + "]"
	default:
		return p.mask
	}
}

// hashString creates a simple hash for PII correlation.
func hashString(s string) string {
	// Simple FNV-1a hash for correlation purposes
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= 1099511628211
	}
	return strings.ToUpper(strings.Replace(
		strings.Replace(
			strings.TrimPrefix(strings.ToUpper(string(rune(hash))), "0X"),
			"-", "", -1),
		"+", "", -1))[:8]
}

// CheckInput checks if input contains PII (for input blocking scenarios).
func (f *PIIFilter) CheckInput(ctx context.Context, input string) CheckResult {
	if input == "" {
		return CheckResult{Blocked: false}
	}

	for _, p := range f.patterns {
		if !f.enabledPII[p.piiType] {
			continue
		}

		select {
		case <-ctx.Done():
			return CheckResult{Blocked: false}
		default:
		}

		if p.pattern.MatchString(input) {
			return CheckResult{
				Blocked:     true,
				Reason:      "PII detected in input: " + string(p.piiType),
				GuardrailID: f.ID(),
				Confidence:  1.0,
				Metadata: map[string]any{
					"pii_type": string(p.piiType),
				},
			}
		}
	}

	return CheckResult{Blocked: false}
}

// WithPIIFilter returns an option that adds PII filtering to output.
func WithPIIFilter(mode PIIFilterMode, opts ...PIIFilterOption) Option {
	return func(g *Guardrails) {
		filter := NewPIIFilter(mode, opts...)
		g.outputFilters = append(g.outputFilters, filter)
	}
}

// WithPIIInputChecker returns an option that blocks input containing PII.
func WithPIIInputChecker(opts ...PIIFilterOption) Option {
	return func(g *Guardrails) {
		filter := NewPIIFilter(PIIFilterMask, opts...) // Mode doesn't matter for input checking
		g.inputCheckers = append(g.inputCheckers, filter)
	}
}
