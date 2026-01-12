package discovery

import (
	"strings"

	"github.com/jllopis/kairos/pkg/config"
)

// BuildProviders builds discovery providers based on config and order.
func BuildProviders(cfg *config.Config, baseURLs []string) []Provider {
	order := ProviderOrder(nil)
	if cfg != nil {
		order = ProviderOrder(cfg.Discovery.Order)
	}
	providers := make([]Provider, 0, len(order))
	for _, item := range order {
		switch strings.ToLower(item) {
		case "config":
			providers = append(providers, NewConfigProvider(cfg))
		case "well_known", "well-known":
			providers = append(providers, NewWellKnownProvider(baseURLs))
		case "registry":
			if cfg != nil && strings.TrimSpace(cfg.Discovery.RegistryURL) != "" {
				provider := NewRegistryProvider(cfg.Discovery.RegistryURL)
				provider.AuthToken = strings.TrimSpace(cfg.Discovery.RegistryToken)
				providers = append(providers, provider)
			}
		}
	}
	return providers
}
