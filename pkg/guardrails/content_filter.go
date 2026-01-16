// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package guardrails

import (
	"context"
	"regexp"
	"strings"
)

// ContentCategory represents a category of content to filter.
type ContentCategory string

const (
	ContentCategoryProfanity  ContentCategory = "profanity"
	ContentCategoryViolence   ContentCategory = "violence"
	ContentCategoryHate       ContentCategory = "hate"
	ContentCategorySexual     ContentCategory = "sexual"
	ContentCategoryDangerous  ContentCategory = "dangerous"
	ContentCategorySelfHarm   ContentCategory = "self_harm"
	ContentCategoryIllegal    ContentCategory = "illegal"
	ContentCategoryMedical    ContentCategory = "medical_advice"
	ContentCategoryFinancial  ContentCategory = "financial_advice"
	ContentCategoryMalware    ContentCategory = "malware"
	ContentCategoryPhishing   ContentCategory = "phishing"
	ContentCategorySpam       ContentCategory = "spam"
)

// contentPattern defines patterns for a content category.
type contentPattern struct {
	category ContentCategory
	patterns []*regexp.Regexp
	keywords []string
}

// ContentFilter blocks or flags content based on category patterns.
type ContentFilter struct {
	categories       map[ContentCategory]contentPattern
	enabledCategories map[ContentCategory]bool
	blockMode        bool // true = block, false = flag only
}

// ContentFilterOption configures the content filter.
type ContentFilterOption func(*ContentFilter)

// Default content patterns (conservative, English-focused)
// In production, consider using ML-based classifiers for better accuracy.
var defaultContentPatterns = map[ContentCategory]struct {
	patterns []string
	keywords []string
}{
	ContentCategoryDangerous: {
		patterns: []string{
			`(?i)how\s+to\s+(make|build|create)\s+(a\s+)?(bomb|explosive|weapon)`,
			`(?i)instructions?\s+(for|to)\s+(making|building)\s+(bombs?|explosives?|weapons?)`,
			`(?i)synthesize\s+(drugs?|chemicals?)`,
		},
		keywords: []string{
			"ricin", "sarin", "anthrax", "weaponize",
		},
	},
	ContentCategorySelfHarm: {
		patterns: []string{
			`(?i)how\s+to\s+(commit\s+)?suicide`,
			`(?i)best\s+way\s+to\s+(kill|harm)\s+(myself|yourself)`,
			`(?i)methods?\s+of\s+self[- ]?harm`,
		},
		keywords: []string{},
	},
	ContentCategoryIllegal: {
		patterns: []string{
			`(?i)how\s+to\s+hack\s+(into|someone)`,
			`(?i)how\s+to\s+(buy|sell)\s+(drugs|stolen)`,
			`(?i)bypass\s+(security|authentication)`,
			`(?i)crack\s+(password|software|license)`,
		},
		keywords: []string{},
	},
	ContentCategoryMalware: {
		patterns: []string{
			`(?i)write\s+(a\s+)?(virus|malware|ransomware|trojan)`,
			`(?i)create\s+(a\s+)?(keylogger|botnet|rootkit)`,
			`(?i)exploit\s+(code|vulnerability)`,
		},
		keywords: []string{
			"shellcode", "reverse shell", "payload injection",
		},
	},
	ContentCategoryPhishing: {
		patterns: []string{
			`(?i)create\s+(a\s+)?phishing\s+(page|email|site)`,
			`(?i)spoof\s+(email|website|identity)`,
			`(?i)social\s+engineer`,
		},
		keywords: []string{},
	},
	ContentCategoryMedical: {
		patterns: []string{
			`(?i)prescribe\s+me`,
			`(?i)what\s+medication\s+should\s+I\s+take`,
			`(?i)diagnose\s+(my|this)\s+(condition|symptoms?)`,
		},
		keywords: []string{},
	},
	ContentCategoryFinancial: {
		patterns: []string{
			`(?i)should\s+I\s+(buy|sell|invest)\s+`,
			`(?i)give\s+me\s+financial\s+advice`,
			`(?i)is\s+this\s+(stock|crypto)\s+a\s+good\s+(buy|investment)`,
		},
		keywords: []string{},
	},
}

// NewContentFilter creates a new content filter.
func NewContentFilter(categories ...ContentCategory) *ContentFilter {
	f := &ContentFilter{
		categories:       make(map[ContentCategory]contentPattern),
		enabledCategories: make(map[ContentCategory]bool),
		blockMode:        true,
	}

	// Compile patterns for requested categories
	for _, cat := range categories {
		if def, ok := defaultContentPatterns[cat]; ok {
			cp := contentPattern{
				category: cat,
				patterns: make([]*regexp.Regexp, 0, len(def.patterns)),
				keywords: def.keywords,
			}
			for _, p := range def.patterns {
				if re, err := regexp.Compile(p); err == nil {
					cp.patterns = append(cp.patterns, re)
				}
			}
			f.categories[cat] = cp
			f.enabledCategories[cat] = true
		}
	}

	return f
}

// WithAllContentCategories enables all default content categories.
func WithAllContentCategories() ContentFilterOption {
	return func(f *ContentFilter) {
		for cat, def := range defaultContentPatterns {
			cp := contentPattern{
				category: cat,
				patterns: make([]*regexp.Regexp, 0, len(def.patterns)),
				keywords: def.keywords,
			}
			for _, p := range def.patterns {
				if re, err := regexp.Compile(p); err == nil {
					cp.patterns = append(cp.patterns, re)
				}
			}
			f.categories[cat] = cp
			f.enabledCategories[cat] = true
		}
	}
}

// WithCustomContentPattern adds a custom pattern to a category.
func WithCustomContentPattern(category ContentCategory, pattern string) ContentFilterOption {
	return func(f *ContentFilter) {
		if re, err := regexp.Compile(pattern); err == nil {
			cp := f.categories[category]
			cp.category = category
			cp.patterns = append(cp.patterns, re)
			f.categories[category] = cp
			f.enabledCategories[category] = true
		}
	}
}

// WithContentKeywords adds keywords to a category.
func WithContentKeywords(category ContentCategory, keywords ...string) ContentFilterOption {
	return func(f *ContentFilter) {
		cp := f.categories[category]
		cp.category = category
		cp.keywords = append(cp.keywords, keywords...)
		f.categories[category] = cp
		f.enabledCategories[category] = true
	}
}

// WithFlagOnly configures the filter to flag but not block content.
func WithFlagOnly() ContentFilterOption {
	return func(f *ContentFilter) {
		f.blockMode = false
	}
}

// ID returns the guardrail identifier.
func (f *ContentFilter) ID() string {
	return "content-filter"
}

// CheckInput analyzes input for disallowed content categories.
func (f *ContentFilter) CheckInput(ctx context.Context, input string) CheckResult {
	if input == "" {
		return CheckResult{Blocked: false}
	}

	normalized := strings.ToLower(input)

	for cat, cp := range f.categories {
		if !f.enabledCategories[cat] {
			continue
		}

		select {
		case <-ctx.Done():
			return CheckResult{Blocked: false}
		default:
		}

		// Check patterns
		for _, pattern := range cp.patterns {
			if pattern.MatchString(normalized) {
				return CheckResult{
					Blocked:     f.blockMode,
					Reason:      "content policy violation: " + string(cat),
					GuardrailID: f.ID(),
					Confidence:  0.9,
					Metadata: map[string]any{
						"category": string(cat),
						"type":     "pattern",
					},
				}
			}
		}

		// Check keywords
		for _, keyword := range cp.keywords {
			if strings.Contains(normalized, strings.ToLower(keyword)) {
				return CheckResult{
					Blocked:     f.blockMode,
					Reason:      "content policy violation: " + string(cat),
					GuardrailID: f.ID(),
					Confidence:  0.8,
					Metadata: map[string]any{
						"category": string(cat),
						"type":     "keyword",
						"keyword":  keyword,
					},
				}
			}
		}
	}

	return CheckResult{Blocked: false}
}

// WithContentFilter returns an option that adds content filtering.
func WithContentFilter(categories ...ContentCategory) Option {
	return func(g *Guardrails) {
		filter := NewContentFilter(categories...)
		g.inputCheckers = append(g.inputCheckers, filter)
	}
}

// WithContentFilterOptions returns an option with additional configuration.
func WithContentFilterOptions(categories []ContentCategory, opts ...ContentFilterOption) Option {
	return func(g *Guardrails) {
		filter := NewContentFilter(categories...)
		for _, opt := range opts {
			opt(filter)
		}
		g.inputCheckers = append(g.inputCheckers, filter)
	}
}
