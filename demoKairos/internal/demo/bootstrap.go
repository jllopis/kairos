package demo

import (
	"fmt"
	"os"
	"strings"

	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/llm"
	"github.com/jllopis/kairos/pkg/telemetry"
)

// LoadConfig loads Kairos configuration from the provided path or defaults.
func LoadConfig(path string) (*config.Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// InitTelemetry initializes telemetry using config settings and verbosity.
func InitTelemetry(service string, cfg *config.Config, verbose bool) (telemetry.ShutdownFunc, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	exporter := strings.TrimSpace(cfg.Telemetry.Exporter)
	if !verbose && (exporter == "" || strings.EqualFold(exporter, "stdout")) {
		exporter = "none"
	}
	return telemetry.InitWithConfig(service, "demo", telemetry.Config{
		Exporter:           exporter,
		OTLPEndpoint:       cfg.Telemetry.OTLPEndpoint,
		OTLPInsecure:       cfg.Telemetry.OTLPInsecure,
		OTLPTimeoutSeconds: cfg.Telemetry.OTLPTimeoutSeconds,
	})
}

// NewLLMProvider builds an LLM provider from config.
func NewLLMProvider(cfg *config.Config) (llm.Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	provider := strings.ToLower(strings.TrimSpace(cfg.LLM.Provider))
	switch provider {
	case "", "ollama":
		baseURL := strings.TrimSpace(os.Getenv("OLLAMA_URL"))
		return llm.NewOllama(baseURL), nil
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", cfg.LLM.Provider)
	}
}
