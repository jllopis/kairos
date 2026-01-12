package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
)

func resetKoanf(t *testing.T) {
	t.Helper()
	k = koanf.New(".")
}

func TestLoadWithCLIOverrides(t *testing.T) {
	resetKoanf(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	content := []byte(`{
  "llm": {"provider": "ollama", "model": "model-a"},
  "telemetry": {"exporter": "stdout"}
}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.Setenv("KAIROS_LLM_PROVIDER", "openai"); err != nil {
		t.Fatalf("set env: %v", err)
	}
	defer os.Unsetenv("KAIROS_LLM_PROVIDER")

	cfg, err := LoadWithCLI([]string{
		"--config", path,
		"--set", "llm.provider=anthropic",
		"--set", "memory.enabled=true",
		"--set", "telemetry.otlp_timeout_seconds=12",
		"--set", "runtime.approval_sweep_interval_seconds=30",
		`--set`, `mcp.servers={"demo":{"transport":"http","url":"http://localhost:8080"}}`,
	})
	if err != nil {
		t.Fatalf("LoadWithCLI failed: %v", err)
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Fatalf("expected cli override provider, got %s", cfg.LLM.Provider)
	}
	if cfg.Memory.Enabled != true {
		t.Fatalf("expected memory.enabled=true")
	}
	if cfg.Telemetry.OTLPTimeoutSeconds != 12 {
		t.Fatalf("expected telemetry timeout override")
	}
	if cfg.Runtime.ApprovalSweepIntervalSeconds != 30 {
		t.Fatalf("expected runtime sweep interval override")
	}
	server, ok := cfg.MCP.Servers["demo"]
	if !ok {
		t.Fatalf("expected demo MCP server override")
	}
	if server.URL != "http://localhost:8080" {
		t.Fatalf("unexpected MCP server url: %s", server.URL)
	}
}

func TestParseCLIOverridesErrors(t *testing.T) {
	resetKoanf(t)
	if _, _, err := parseCLIOverrides([]string{"--config"}); err == nil {
		t.Fatalf("expected error for missing --config value")
	}
	if _, _, err := parseCLIOverrides([]string{"--set"}); err == nil {
		t.Fatalf("expected error for missing --set value")
	}
	if _, _, err := parseCLIOverrides([]string{"--set", "invalid"}); err == nil {
		t.Fatalf("expected error for invalid --set value")
	}
}
