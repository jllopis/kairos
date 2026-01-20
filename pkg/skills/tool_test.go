// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jllopis/kairos/pkg/llm"
)

func TestSkillTool_Name(t *testing.T) {
	spec := SkillSpec{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "Do something when activated.",
	}

	tool := NewSkillTool(spec)

	if tool.Name() != "test-skill" {
		t.Errorf("expected name 'test-skill', got %q", tool.Name())
	}
}

func TestSkillTool_Activate(t *testing.T) {
	spec := SkillSpec{
		Name:        "test-skill",
		Description: "A test skill",
		Body:        "These are the instructions for the skill.",
	}

	tool := NewSkillTool(spec)

	if tool.IsActivated() {
		t.Error("skill should not be activated before Call")
	}

	result, err := tool.Call(context.Background(), nil)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if !tool.IsActivated() {
		t.Error("skill should be activated after Call")
	}

	resp, ok := result.(*SkillResponse)
	if !ok {
		t.Fatalf("expected *SkillResponse, got %T", result)
	}

	if resp.Name != "test-skill" {
		t.Errorf("expected name 'test-skill', got %q", resp.Name)
	}

	if resp.Instructions != "These are the instructions for the skill." {
		t.Errorf("unexpected instructions: %q", resp.Instructions)
	}
}

func TestSkillTool_ActivateWithJSON(t *testing.T) {
	spec := SkillSpec{
		Name:        "json-skill",
		Description: "Skill with JSON input",
		Body:        "JSON instructions here.",
	}

	tool := NewSkillTool(spec)

	result, err := tool.Call(context.Background(), `{"action": "activate"}`)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	resp, ok := result.(*SkillResponse)
	if !ok {
		t.Fatalf("expected *SkillResponse, got %T", result)
	}

	if resp.Instructions != "JSON instructions here." {
		t.Errorf("unexpected instructions: %q", resp.Instructions)
	}
}

func TestSkillTool_LoadResource(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "resource-skill")
	scriptsDir := filepath.Join(skillDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create SKILL.md
	skillContent := `---
name: resource-skill
description: A skill with resources.
---

Use this skill for resource loading.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	// Create a script
	scriptContent := "#!/bin/bash\necho 'Hello from script'"
	if err := os.WriteFile(filepath.Join(scriptsDir, "extract.sh"), []byte(scriptContent), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	spec, err := LoadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("load skill: %v", err)
	}

	tool := NewSkillTool(spec)

	// Test load_resource action
	result, err := tool.Call(context.Background(), map[string]any{
		"action":   "load_resource",
		"resource": "scripts/extract.sh",
	})
	if err != nil {
		t.Fatalf("load_resource failed: %v", err)
	}

	content, ok := result.(string)
	if !ok {
		t.Fatalf("expected string, got %T", result)
	}

	if content != scriptContent {
		t.Errorf("unexpected content: %q", content)
	}
}

func TestSkillTool_ListResources(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "list-skill")
	scriptsDir := filepath.Join(skillDir, "scripts")
	refsDir := filepath.Join(skillDir, "references")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts: %v", err)
	}
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		t.Fatalf("mkdir refs: %v", err)
	}

	// Create SKILL.md
	skillContent := `---
name: list-skill
description: A skill with resources to list.
---

Instructions here.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	// Create resources
	if err := os.WriteFile(filepath.Join(scriptsDir, "script1.py"), []byte("# python"), 0o644); err != nil {
		t.Fatalf("write script1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(refsDir, "REFERENCE.md"), []byte("# ref"), 0o644); err != nil {
		t.Fatalf("write ref: %v", err)
	}

	spec, err := LoadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("load skill: %v", err)
	}

	tool := NewSkillTool(spec)

	result, err := tool.Call(context.Background(), map[string]any{
		"action": "list_resources",
	})
	if err != nil {
		t.Fatalf("list_resources failed: %v", err)
	}

	resources, ok := result.([]string)
	if !ok {
		t.Fatalf("expected []string, got %T", result)
	}

	if len(resources) != 2 {
		t.Errorf("expected 2 resources, got %d: %v", len(resources), resources)
	}
}

func TestSkillTool_LoadResourceSecurityTraversal(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "secure-skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Create a file outside skill dir
	outsideContent := "secret data"
	if err := os.WriteFile(filepath.Join(dir, "secret.txt"), []byte(outsideContent), 0o644); err != nil {
		t.Fatalf("write secret: %v", err)
	}

	skillContent := `---
name: secure-skill
description: Security test skill.
---

Instructions.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	spec, err := LoadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("load skill: %v", err)
	}

	tool := NewSkillTool(spec)

	// Attempt directory traversal
	_, err = tool.Call(context.Background(), map[string]any{
		"action":   "load_resource",
		"resource": "../secret.txt",
	})
	if err == nil {
		t.Error("expected error for directory traversal, got nil")
	}
}

func TestSkillTool_ToolDefinition(t *testing.T) {
	spec := SkillSpec{
		Name:        "def-skill",
		Description: "A skill with tool definition.",
		Body:        "Instructions for the skill.",
	}

	tool := NewSkillTool(spec)
	def := tool.ToolDefinition()

	if def.Type != llm.ToolTypeFunction {
		t.Errorf("expected type function, got %v", def.Type)
	}

	if def.Function.Name != "def-skill" {
		t.Errorf("expected name 'def-skill', got %q", def.Function.Name)
	}

	if def.Function.Description != "A skill with tool definition." {
		t.Errorf("unexpected description: %q", def.Function.Description)
	}
}

func TestLoadToolsFromDir(t *testing.T) {
	dir := t.TempDir()

	// Create two skills
	for _, name := range []string{"skill-a", "skill-b"} {
		skillDir := filepath.Join(dir, name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		content := `---
name: ` + name + `
description: Test skill ` + name + `.
---

Instructions for ` + name + `.
`
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write SKILL.md for %s: %v", name, err)
		}
	}

	tools, err := LoadToolsFromDir(dir)
	if err != nil {
		t.Fatalf("LoadToolsFromDir: %v", err)
	}

	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}

	names := make(map[string]bool)
	for _, tool := range tools {
		names[tool.Name()] = true
	}

	if !names["skill-a"] || !names["skill-b"] {
		t.Errorf("expected skill-a and skill-b, got %v", names)
	}
}
