package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.LLM.Provider != "ollama" {
		t.Errorf("expected default provider ollama, got %s", cfg.LLM.Provider)
	}
	if cfg.LLM.Model != "qwen2.5-coder:7b-instruct-q5_K_M" {
		t.Errorf("expected default model qwen2.5..., got %s", cfg.LLM.Model)
	}
}

func TestLoadEnv(t *testing.T) {
	os.Setenv("KAIROS_LLM_PROVIDER", "openai")
	defer os.Unsetenv("KAIROS_LLM_PROVIDER")

	k.Delete("llm.provider") // Clear previous state if any (koanf global instance)
	// Note: Since we use a global 'k', we should be careful.
	// In a real app we might want to avoid global state, but for this refactor it's fine for now.
	// However, Load re-initializes defaults. Let's see if it works.

	// Actually, Load() sets defaults on 'k' every time.

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.LLM.Provider != "openai" {
		t.Errorf("expected provider openai from env, got %s", cfg.LLM.Provider)
	}
}

func TestLoadWithProfile(t *testing.T) {
	// Create temp directory with config files
	tmpDir := t.TempDir()

	// Base config
	baseConfig := `
llm:
  provider: "ollama"
  model: "llama3.1"
log:
  level: "info"
`
	basePath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(basePath, []byte(baseConfig), 0644); err != nil {
		t.Fatalf("failed to write base config: %v", err)
	}

	// Dev profile override
	devConfig := `
llm:
  provider: "mock"
log:
  level: "debug"
`
	devPath := filepath.Join(tmpDir, "config.dev.yaml")
	if err := os.WriteFile(devPath, []byte(devConfig), 0644); err != nil {
		t.Fatalf("failed to write dev config: %v", err)
	}

	// Prod profile override
	prodConfig := `
llm:
  provider: "openai"
log:
  level: "warn"
`
	prodPath := filepath.Join(tmpDir, "config.prod.yaml")
	if err := os.WriteFile(prodPath, []byte(prodConfig), 0644); err != nil {
		t.Fatalf("failed to write prod config: %v", err)
	}

	tests := []struct {
		name         string
		profile      string
		wantProvider string
		wantLogLevel string
		wantModel    string // Should inherit from base when not overridden
	}{
		{
			name:         "no profile - base only",
			profile:      "",
			wantProvider: "ollama",
			wantLogLevel: "info",
			wantModel:    "llama3.1",
		},
		{
			name:         "dev profile",
			profile:      "dev",
			wantProvider: "mock",
			wantLogLevel: "debug",
			wantModel:    "llama3.1", // Not overridden in dev
		},
		{
			name:         "prod profile",
			profile:      "prod",
			wantProvider: "openai",
			wantLogLevel: "warn",
			wantModel:    "llama3.1", // Not overridden in prod
		},
		{
			name:         "nonexistent profile - falls back to base",
			profile:      "staging",
			wantProvider: "ollama",
			wantLogLevel: "info",
			wantModel:    "llama3.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadWithProfile(basePath, tc.profile)
			if err != nil {
				t.Fatalf("LoadWithProfile failed: %v", err)
			}

			if cfg.LLM.Provider != tc.wantProvider {
				t.Errorf("provider: got %s, want %s", cfg.LLM.Provider, tc.wantProvider)
			}
			if cfg.Log.Level != tc.wantLogLevel {
				t.Errorf("log level: got %s, want %s", cfg.Log.Level, tc.wantLogLevel)
			}
			if cfg.LLM.Model != tc.wantModel {
				t.Errorf("model: got %s, want %s", cfg.LLM.Model, tc.wantModel)
			}
		})
	}
}

func TestLoadWithCLIProfile(t *testing.T) {
	// Create temp directory with config files
	tmpDir := t.TempDir()

	baseConfig := `
llm:
  provider: "ollama"
`
	basePath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(basePath, []byte(baseConfig), 0644); err != nil {
		t.Fatalf("failed to write base config: %v", err)
	}

	devConfig := `
llm:
  provider: "mock"
`
	devPath := filepath.Join(tmpDir, "config.dev.yaml")
	if err := os.WriteFile(devPath, []byte(devConfig), 0644); err != nil {
		t.Fatalf("failed to write dev config: %v", err)
	}

	tests := []struct {
		name         string
		args         []string
		wantProvider string
	}{
		{
			name:         "profile flag",
			args:         []string{"--config", basePath, "--profile", "dev"},
			wantProvider: "mock",
		},
		{
			name:         "env flag alias",
			args:         []string{"--config", basePath, "--env", "dev"},
			wantProvider: "mock",
		},
		{
			name:         "profile with equals",
			args:         []string{"--config=" + basePath, "--profile=dev"},
			wantProvider: "mock",
		},
		{
			name:         "env with equals",
			args:         []string{"--config=" + basePath, "--env=dev"},
			wantProvider: "mock",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := LoadWithCLI(tc.args)
			if err != nil {
				t.Fatalf("LoadWithCLI failed: %v", err)
			}

			if cfg.LLM.Provider != tc.wantProvider {
				t.Errorf("provider: got %s, want %s", cfg.LLM.Provider, tc.wantProvider)
			}
		})
	}
}

func TestLoadWithCLITelemetryHeaders(t *testing.T) {
	args := []string{
		"--set", "telemetry.exporter=otlp",
		"--set", "telemetry.otlp_endpoint=http://localhost:4317",
		"--set", "telemetry.otlp_headers.x-api-key=secret-token",
		"--set", "telemetry.otlp_headers.x-org-id=org-123",
	}

	cfg, err := LoadWithCLI(args)
	if err != nil {
		t.Fatalf("LoadWithCLI failed: %v", err)
	}

	if cfg.Telemetry.Exporter != "otlp" {
		t.Errorf("expected exporter otlp, got %s", cfg.Telemetry.Exporter)
	}
	if cfg.Telemetry.OTLPEndpoint != "http://localhost:4317" {
		t.Errorf("expected endpoint, got %s", cfg.Telemetry.OTLPEndpoint)
	}

	headers := cfg.Telemetry.OTLPHeaders
	if headers["x-api-key"] != "secret-token" {
		t.Errorf("expected x-api-key=secret-token, got %s", headers["x-api-key"])
	}
	if headers["x-org-id"] != "org-123" {
		t.Errorf("expected x-org-id=org-123, got %s", headers["x-org-id"])
	}
}

func TestLoadWithCLITelemetryBasicAuth(t *testing.T) {
	args := []string{
		"--set", "telemetry.exporter=otlp",
		"--set", "telemetry.otlp_user=admin",
		"--set", "telemetry.otlp_token=password123",
	}

	cfg, err := LoadWithCLI(args)
	if err != nil {
		t.Fatalf("LoadWithCLI failed: %v", err)
	}

	if cfg.Telemetry.OTLPUser != "admin" {
		t.Errorf("expected user admin, got %s", cfg.Telemetry.OTLPUser)
	}
	if cfg.Telemetry.OTLPToken != "password123" {
		t.Errorf("expected token password123, got %s", cfg.Telemetry.OTLPToken)
	}
}

func TestProfileConfigPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create config.dev.yaml
	devPath := filepath.Join(tmpDir, "config.dev.yaml")
	if err := os.WriteFile(devPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create dev config: %v", err)
	}

	basePath := filepath.Join(tmpDir, "config.yaml")

	tests := []struct {
		name     string
		base     string
		profile  string
		wantPath string
	}{
		{
			name:     "existing profile",
			base:     basePath,
			profile:  "dev",
			wantPath: devPath,
		},
		{
			name:     "nonexistent profile",
			base:     basePath,
			profile:  "prod",
			wantPath: "",
		},
		{
			name:     "empty profile",
			base:     basePath,
			profile:  "",
			wantPath: "",
		},
		{
			name:     "empty base",
			base:     "",
			profile:  "dev",
			wantPath: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := profileConfigPath(tc.base, tc.profile)
			if got != tc.wantPath {
				t.Errorf("profileConfigPath(%q, %q) = %q, want %q", tc.base, tc.profile, got, tc.wantPath)
			}
		})
	}
}
