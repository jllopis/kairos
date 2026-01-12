package discovery

import (
	"context"
	"strings"

	"github.com/jllopis/kairos/pkg/config"
)

// ConfigProvider lists agents from configuration.
type ConfigProvider struct {
	Entries []AgentEndpoint
}

// NewConfigProvider builds a provider from config.
func NewConfigProvider(cfg *config.Config) *ConfigProvider {
	provider := &ConfigProvider{}
	if cfg == nil {
		return provider
	}
	for name, agentCfg := range cfg.Agents {
		endpoint := AgentEndpoint{
			Name:         strings.TrimSpace(name),
			AgentCardURL: strings.TrimSpace(agentCfg.AgentCardURL),
			GRPCAddr:     strings.TrimSpace(agentCfg.GRPCAddr),
			HTTPURL:      strings.TrimSpace(agentCfg.HTTPURL),
			Labels:       cloneLabels(agentCfg.Labels),
		}
		provider.Entries = append(provider.Entries, endpoint)
	}
	return provider
}

// List returns configured endpoints.
func (p *ConfigProvider) List(_ context.Context) ([]AgentEndpoint, error) {
	if p == nil {
		return nil, nil
	}
	return append([]AgentEndpoint(nil), p.Entries...), nil
}

func cloneLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}
	out := make(map[string]string, len(labels))
	for key, value := range labels {
		out[key] = value
	}
	return out
}
