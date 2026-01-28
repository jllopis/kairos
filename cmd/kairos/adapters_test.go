// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"
)

func TestAdaptersRegistry(t *testing.T) {
	if len(adaptersRegistry) == 0 {
		t.Error("adapters registry should not be empty")
	}

	// Check we have at least one of each type
	types := map[string]bool{}
	for _, a := range adaptersRegistry {
		types[a.Type] = true
	}

	expectedTypes := []string{"llm", "memory", "mcp", "a2a", "telemetry"}
	for _, et := range expectedTypes {
		if !types[et] {
			t.Errorf("expected adapter type %q not found", et)
		}
	}
}

func TestAdapterHasRequiredFields(t *testing.T) {
	for _, a := range adaptersRegistry {
		if a.Name == "" {
			t.Error("adapter name should not be empty")
		}
		if a.Type == "" {
			t.Errorf("adapter %q type should not be empty", a.Name)
		}
		if a.Description == "" {
			t.Errorf("adapter %q description should not be empty", a.Name)
		}
	}
}

func TestFilterAdaptersByType(t *testing.T) {
	filtered := make([]Adapter, 0)
	for _, a := range adaptersRegistry {
		if a.Type == "llm" {
			filtered = append(filtered, a)
		}
	}

	if len(filtered) == 0 {
		t.Error("expected at least one LLM adapter")
	}

	for _, a := range filtered {
		if a.Type != "llm" {
			t.Errorf("filtered adapter %q has wrong type: %s", a.Name, a.Type)
		}
	}
}
