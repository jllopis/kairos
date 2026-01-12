package discovery

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jllopis/kairos/pkg/config"
)

const defaultHeartbeat = 10 * time.Second

// StartAutoRegister registers the endpoint and refreshes it on interval.
func StartAutoRegister(ctx context.Context, provider *RegistryProvider, endpoint AgentEndpoint, interval time.Duration) (context.CancelFunc, error) {
	if provider == nil || provider.BaseURL == "" {
		return nil, errors.New("registry provider not configured")
	}
	if normalizeKey(endpoint.AgentCardURL, endpoint.Name) == "" {
		return nil, errors.New("agent endpoint missing name or agent_card_url")
	}
	if interval <= 0 {
		interval = defaultHeartbeat
	}
	ctx, cancel := context.WithCancel(ctx)
	logger := slog.Default()

	register := func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := provider.Register(ctx, endpoint); err != nil {
			logger.Warn("discovery.registry.register.failed", slog.String("error", err.Error()))
		}
	}

	register()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				register()
			}
		}
	}()

	return cancel, nil
}

// StartAutoRegisterFromConfig wires auto-register from config settings.
func StartAutoRegisterFromConfig(ctx context.Context, cfg *config.Config, endpoint AgentEndpoint) (context.CancelFunc, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}
	if !cfg.Discovery.AutoRegister {
		return nil, nil
	}
	provider := NewRegistryProvider(cfg.Discovery.RegistryURL)
	provider.AuthToken = cfg.Discovery.RegistryToken
	interval := time.Duration(cfg.Discovery.HeartbeatSeconds) * time.Second
	return StartAutoRegister(ctx, provider, endpoint, interval)
}
