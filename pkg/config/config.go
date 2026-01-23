// Package config loads and normalizes runtime configuration.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config is the root configuration for the Kairos runtime.
type Config struct {
	Log        LogConfig                      `koanf:"log"`
	LLM        LLMConfig                      `koanf:"llm"`
	Agent      AgentConfig                    `koanf:"agent"`
	Agents     map[string]AgentOverrideConfig `koanf:"agents"`
	Memory     MemoryConfig                   `koanf:"memory"`
	MCP        MCPConfig                      `koanf:"mcp"`
	Discovery  DiscoveryConfig                `koanf:"discovery"`
	Telemetry  TelemetryConfig                `koanf:"telemetry"`
	Runtime    RuntimeConfig                  `koanf:"runtime"`
	Governance GovernanceConfig               `koanf:"governance"`
}

// LogConfig controls logging output.
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"` // json, text
}

// LLMConfig selects the LLM provider and model settings.
type LLMConfig struct {
	Provider string `koanf:"provider"` // openai, anthropic, ollama
	Model    string `koanf:"model"`
	BaseURL  string `koanf:"base_url"`
	APIKey   string `koanf:"api_key"`
}

// AgentConfig defines global defaults for agent behavior.
type AgentConfig struct {
	DisableActionFallback bool `koanf:"disable_action_fallback"`
	WarnOnActionFallback  bool `koanf:"warn_on_action_fallback"`
}

// AgentOverrideConfig overrides per-agent settings.
type AgentOverrideConfig struct {
	DisableActionFallback *bool             `koanf:"disable_action_fallback"`
	WarnOnActionFallback  *bool             `koanf:"warn_on_action_fallback"`
	AgentCardURL          string            `koanf:"agent_card_url"`
	GRPCAddr              string            `koanf:"grpc_addr"`
	HTTPURL               string            `koanf:"http_url"`
	Labels                map[string]string `koanf:"labels"`
}

// MemoryConfig configures memory backends.
type MemoryConfig struct {
	Enabled          bool   `koanf:"enabled"`
	Provider         string `koanf:"provider"` // vector, inmemory
	QdrantAddr       string `koanf:"qdrant_addr"`
	EmbedderProvider string `koanf:"embedder_provider"` // ollama
	EmbedderBaseURL  string `koanf:"embedder_base_url"`
	EmbedderModel    string `koanf:"embedder_model"`
}

// MCPConfig defines MCP server connections.
type MCPConfig struct {
	Servers map[string]MCPServerConfig `koanf:"servers"`
}

// DiscoveryConfig defines agent discovery options.
type DiscoveryConfig struct {
	Order            []string `koanf:"order"`
	RegistryURL      string   `koanf:"registry_url"`
	RegistryToken    string   `koanf:"registry_token"`
	AutoRegister     bool     `koanf:"auto_register"`
	HeartbeatSeconds int      `koanf:"heartbeat_seconds"`
}

// MCPServerConfig describes an MCP server endpoint and client settings.
type MCPServerConfig struct {
	Command         string   `koanf:"command"`
	Args            []string `koanf:"args"`
	ProtocolVersion string   `koanf:"protocol_version"`
	Transport       string   `koanf:"transport"` // stdio, http
	URL             string   `koanf:"url"`
	TimeoutSeconds  *int     `koanf:"timeout_seconds"`
	RetryCount      *int     `koanf:"retry_count"`
	RetryBackoffMs  *int     `koanf:"retry_backoff_ms"`
	CacheTTLSeconds *int     `koanf:"cache_ttl_seconds"`
}

// TelemetryConfig configures OpenTelemetry exporters.
type TelemetryConfig struct {
	Exporter           string            `koanf:"exporter"` // stdout, otlp
	OTLPEndpoint       string            `koanf:"otlp_endpoint"`
	OTLPInsecure       bool              `koanf:"otlp_insecure"`
	OTLPTimeoutSeconds int               `koanf:"otlp_timeout_seconds"`
	OTLPHeaders        map[string]string `koanf:"otlp_headers"`
	OTLPUser           string            `koanf:"otlp_user"`
	OTLPToken          string            `koanf:"otlp_token"`
}

// RuntimeConfig configures runtime-level behaviors.
type RuntimeConfig struct {
	ApprovalSweepIntervalSeconds int `koanf:"approval_sweep_interval_seconds"`
	ApprovalSweepTimeoutSeconds  int `koanf:"approval_sweep_timeout_seconds"`
}

// GovernanceConfig defines policy and instruction loading options.
type GovernanceConfig struct {
	Policies               []PolicyRuleConfig `koanf:"policies"`
	ApprovalTimeoutSeconds int                `koanf:"approval_timeout_seconds"`
}

// PolicyRuleConfig defines a single policy rule.
type PolicyRuleConfig struct {
	ID     string `koanf:"id"`
	Effect string `koanf:"effect"`
	Type   string `koanf:"type"`
	Name   string `koanf:"name"`
	Reason string `koanf:"reason"`
}

// Global k instance
var k = koanf.New(".")

// Load resolves configuration from defaults, files, and environment variables.
func Load(path string) (*Config, error) {
	return loadWithOverrides(path, "", nil)
}

// LoadWithProfile resolves configuration with environment-specific layering.
// It loads the base config file, then merges the profile-specific override file.
//
// Example with profile="dev":
//   - config.yaml (base)
//   - config.dev.yaml (override, merged on top)
//
// Example with profile="prod":
//   - config.yaml (base)
//   - config.prod.yaml (override, merged on top)
//
// Profile-specific files are optional; if not found, only the base is used.
func LoadWithProfile(path, profile string) (*Config, error) {
	return loadWithOverrides(path, profile, nil)
}

// LoadWithCLI resolves configuration and applies CLI overrides from args.
// Supported flags:
// - --config=/path/to/settings.json
// - --profile=dev (or --env=dev)
// - --set key=value (repeatable)
func LoadWithCLI(args []string) (*Config, error) {
	path, profile, overrides, err := parseCLIOverrides(args)
	if err != nil {
		return nil, err
	}
	return loadWithOverrides(path, profile, overrides)
}

func loadWithOverrides(path, profile string, overrides map[string]any) (*Config, error) {
	// Defaults
	k.Set("log.level", "info")
	k.Set("log.format", "text")
	k.Set("llm.provider", "ollama")
	k.Set("llm.model", "qwen2.5-coder:7b-instruct-q5_K_M")
	k.Set("llm.base_url", "http://localhost:11434")
	k.Set("agent.disable_action_fallback", true)
	k.Set("agent.warn_on_action_fallback", false)

	k.Set("memory.enabled", false)
	k.Set("memory.provider", "vector")
	k.Set("memory.qdrant_addr", "localhost:6334")
	k.Set("memory.embedder_provider", "ollama")
	k.Set("memory.embedder_base_url", "http://localhost:11434")
	k.Set("memory.embedder_model", "nomic-embed-text") // Default embedding model

	k.Set("telemetry.exporter", "stdout")
	k.Set("telemetry.otlp_endpoint", "")
	k.Set("telemetry.otlp_insecure", true)
	k.Set("telemetry.otlp_timeout_seconds", 10)
	k.Set("telemetry.otlp_headers", map[string]string{})
	k.Set("telemetry.otlp_user", "")
	k.Set("telemetry.otlp_token", "")

	k.Set("runtime.approval_sweep_interval_seconds", 0)
	k.Set("runtime.approval_sweep_timeout_seconds", 0)

	k.Set("governance.policies", []PolicyRuleConfig{})
	k.Set("governance.approval_timeout_seconds", 0)
	k.Set("discovery.order", []string{})
	k.Set("discovery.auto_register", false)
	k.Set("discovery.heartbeat_seconds", 0)

	// 1. Load from file
	configPath := path
	if configPath == "" {
		configPath = defaultConfigPath()
	}
	if configPath != "" {
		if err := loadFromFile(configPath); err != nil {
			return nil, err
		}
	}

	// 2. Load profile-specific override file (config.dev.yaml, config.prod.yaml, etc.)
	if profile != "" && configPath != "" {
		profilePath := profileConfigPath(configPath, profile)
		if profilePath != "" {
			if err := loadFromFile(profilePath); err != nil {
				return nil, err
			}
		}
	}

	// 3. Load from ENV (KAIROS_LLM_PROVIDER -> llm.provider)
	if err := k.Load(env.Provider("KAIROS_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "KAIROS_")), "_", ".", -1)
	}), nil); err != nil {
		return nil, err
	}

	if len(overrides) > 0 {
		for key, value := range overrides {
			_ = k.Set(key, value)
		}
	}

	normalizeMCPServers()
	normalizeMCPServerTransport()
	normalizeTelemetryConfig()

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func parseCLIOverrides(args []string) (string, string, map[string]any, error) {
	overrides := make(map[string]any)
	var path, profile string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if arg == "--config" {
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("missing value for --config")
			}
			path = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--config=") {
			path = strings.TrimPrefix(arg, "--config=")
			continue
		}
		// Support both --profile and --env for environment layering
		if arg == "--profile" || arg == "--env" {
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("missing value for %s", arg)
			}
			profile = args[i+1]
			i++
			continue
		}
		if strings.HasPrefix(arg, "--profile=") {
			profile = strings.TrimPrefix(arg, "--profile=")
			continue
		}
		if strings.HasPrefix(arg, "--env=") {
			profile = strings.TrimPrefix(arg, "--env=")
			continue
		}
		if arg == "--set" {
			if i+1 >= len(args) {
				return "", "", nil, fmt.Errorf("missing value for --set")
			}
			key, value, err := parseKeyValue(args[i+1])
			if err != nil {
				return "", "", nil, err
			}
			overrides[key] = value
			i++
			continue
		}
		if strings.HasPrefix(arg, "--set=") {
			key, value, err := parseKeyValue(strings.TrimPrefix(arg, "--set="))
			if err != nil {
				return "", "", nil, err
			}
			overrides[key] = value
			continue
		}
	}
	return path, profile, overrides, nil
}

func parseKeyValue(raw string) (string, any, error) {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid --set value %q (expected key=value)", raw)
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", nil, fmt.Errorf("invalid --set key in %q", raw)
	}
	value := parseOverrideValue(strings.TrimSpace(parts[1]))
	return key, value, nil
}

func parseOverrideValue(raw string) any {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "\"") {
		var value any
		if err := json.Unmarshal([]byte(raw), &value); err == nil {
			return value
		}
	}
	if value, err := strconv.ParseBool(raw); err == nil {
		return value
	}
	if value, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return value
	}
	if value, err := strconv.ParseFloat(raw, 64); err == nil {
		return value
	}
	return raw
}

// AgentConfigFor returns the effective agent config for a specific agent id.
func (c *Config) AgentConfigFor(id string) AgentConfig {
	if c == nil {
		return AgentConfig{}
	}
	cfg := c.Agent
	if c.Agents == nil || id == "" {
		return cfg
	}
	override, ok := c.Agents[id]
	if !ok {
		return cfg
	}
	if override.DisableActionFallback != nil {
		cfg.DisableActionFallback = *override.DisableActionFallback
	}
	if override.WarnOnActionFallback != nil {
		cfg.WarnOnActionFallback = *override.WarnOnActionFallback
	}
	return cfg
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

// profileConfigPath returns the path to a profile-specific config file.
// For "config.yaml" with profile "dev", it returns "config.dev.yaml".
// Returns empty string if the profile config file doesn't exist.
func profileConfigPath(basePath, profile string) string {
	if basePath == "" || profile == "" {
		return ""
	}

	dir := filepath.Dir(basePath)
	base := filepath.Base(basePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Build profile path: config.yaml -> config.dev.yaml
	profilePath := filepath.Join(dir, name+"."+profile+ext)

	if _, err := os.Stat(profilePath); err == nil {
		return profilePath
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

func normalizeMCPServerTransport() {
	raw := k.Get("mcp.servers")
	servers, ok := raw.(map[string]interface{})
	if !ok || len(servers) == 0 {
		return
	}
	updated := false
	for name, entry := range servers {
		payload, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok := payload["transport"]; ok {
			continue
		}
		if transport, ok := payload["type"]; ok {
			payload["transport"] = transport
			servers[name] = payload
			updated = true
		}
	}
	if updated {
		_ = k.Set("mcp.servers", servers)
	}
}

func normalizeTelemetryConfig() {
	if !k.Exists("telemetry.otlp_endpoint") || k.String("telemetry.otlp_endpoint") == "" {
		if raw := k.Get("telemetry.otlp.endpoint"); raw != nil {
			_ = k.Set("telemetry.otlp_endpoint", raw)
		}
	}
	if !k.Exists("telemetry.otlp_insecure") {
		if raw := k.Get("telemetry.otlp.insecure"); raw != nil {
			_ = k.Set("telemetry.otlp_insecure", raw)
		}
	}
	if !k.Exists("telemetry.otlp_timeout_seconds") {
		if raw := k.Get("telemetry.otlp.timeout_seconds"); raw != nil {
			_ = k.Set("telemetry.otlp_timeout_seconds", raw)
		}
	}
	if !k.Exists("telemetry.otlp_headers") || len(k.StringMap("telemetry.otlp_headers")) == 0 {
		if raw := k.Get("telemetry.otlp.headers"); raw != nil {
			_ = k.Set("telemetry.otlp_headers", raw)
		}
	}
	if !k.Exists("telemetry.otlp_user") || k.String("telemetry.otlp_user") == "" {
		if raw := k.Get("telemetry.otlp.user"); raw != nil {
			_ = k.Set("telemetry.otlp_user", raw)
		}
	}
	if !k.Exists("telemetry.otlp_token") || k.String("telemetry.otlp_token") == "" {
		if raw := k.Get("telemetry.otlp.token"); raw != nil {
			_ = k.Set("telemetry.otlp_token", raw)
		}
	}
}
