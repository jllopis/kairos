package config

import (
	"os"
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
