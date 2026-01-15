package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "pdf-processing")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: pdf-processing
description: Extracts text and tables from PDF files.
license: Apache-2.0
compatibility: Requires pdftotext
metadata:
  author: example-org
allowed-tools: Bash(pdf:* ) Bash(ocr:*)
---

Use this skill when dealing with PDFs.
`
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	skill, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if skill.Name != "pdf-processing" {
		t.Fatalf("unexpected name: %s", skill.Name)
	}
	if len(skill.AllowedTools) != 2 {
		t.Fatalf("expected allowed tools, got %v", skill.AllowedTools)
	}
}

func TestLoadDir(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "code-review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: code-review
description: Review code changes.
---
`
	path := filepath.Join(skillDir, "SKILL.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	skills, err := LoadDir(dir)
	if err != nil {
		t.Fatalf("load dir: %v", err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
}
