package config

import (
	"os"
	"path/filepath"
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
	MCP    MCPConfig    `koanf:"mcp"`
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

type MCPConfig struct {
	Servers map[string]MCPServerConfig `koanf:"servers"`
}

type MCPServerConfig struct {
	Command         string   `koanf:"command"`
	Args            []string `koanf:"args"`
	ProtocolVersion string   `koanf:"protocol_version"`
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
		if err := loadFromFile(path); err != nil {
			return nil, err
		}
	} else {
		if defaultPath := defaultConfigPath(); defaultPath != "" {
			if err := loadFromFile(defaultPath); err != nil {
				return nil, err
			}
		}
	}

	// 2. Load from ENV (KAIROS_LLM_PROVIDER -> llm.provider)
	if err := k.Load(env.Provider("KAIROS_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "KAIROS_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	normalizeMCPServers()

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadFromFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.IsDir() {
		return nil
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return k.Load(file.Provider(path), yaml.Parser())
	case ".yaml", ".yml":
		return k.Load(file.Provider(path), yaml.Parser())
	default:
		return k.Load(file.Provider(path), yaml.Parser())
	}
}

func defaultConfigPath() string {
	candidates := []string{
		filepath.Join(".kairos", "settings.json"),
	}
	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".kairos", "settings.json"))
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		candidates = append(candidates, filepath.Join(xdg, "kairos", "settings.json"))
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func normalizeMCPServers() {
	if k.Exists("mcp.servers") {
		return
	}
	raw := k.Get("mcpServers")
	if raw == nil {
		return
	}
	_ = k.Set("mcp.servers", raw)
}
