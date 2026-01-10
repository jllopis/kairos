package config

import (
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Log    LogConfig    `koanf:"log"`
	LLM    LLMConfig    `koanf:"llm"`
	Memory MemoryConfig `koanf:"memory"`
}

type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"` // json, text
}

type LLMConfig struct {
	Provider string `koanf:"provider"` // openai, anthropic, ollama
	Model    string `koanf:"model"`
	BaseURL  string `koanf:"base_url"`
	APIKey   string `koanf:"api_key"`
}

type MemoryConfig struct {
	Enabled          bool   `koanf:"enabled"`
	Provider         string `koanf:"provider"` // vector, inmemory
	QdrantAddr       string `koanf:"qdrant_addr"`
	EmbedderProvider string `koanf:"embedder_provider"` // ollama
	EmbedderBaseURL  string `koanf:"embedder_base_url"`
	EmbedderModel    string `koanf:"embedder_model"`
}

// Global k instance
var k = koanf.New(".")

func Load(path string) (*Config, error) {
	// Defaults
	k.Set("log.level", "info")
	k.Set("log.format", "text")
	k.Set("llm.provider", "ollama")
	k.Set("llm.model", "qwen2.5-coder:7b-instruct-q5_K_M")
	k.Set("llm.base_url", "http://localhost:11434")

	k.Set("memory.enabled", false)
	k.Set("memory.provider", "vector")
	k.Set("memory.qdrant_addr", "localhost:6334")
	k.Set("memory.embedder_provider", "ollama")
	k.Set("memory.embedder_base_url", "http://localhost:11434")
	k.Set("memory.embedder_model", "nomic-embed-text") // Default embedding model

	// 1. Load from file
	if path != "" {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, err
		}
	}

	// 2. Load from ENV (KAIROS_LLM_PROVIDER -> llm.provider)
	if err := k.Load(env.Provider("KAIROS_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "KAIROS_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
