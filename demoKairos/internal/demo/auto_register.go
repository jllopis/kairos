package demo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jllopis/kairos/pkg/config"
	"github.com/jllopis/kairos/pkg/discovery"
)

// AgentEndpoint describes a demo agent registration entry.
type AgentEndpoint struct {
	Name         string
	AgentCardURL string
	GRPCAddr     string
	HTTPURL      string
}

// AgentCardURL builds a base URL for AgentCard discovery.
func AgentCardURL(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, "http://") || strings.HasPrefix(addr, "https://") {
		return strings.TrimRight(addr, "/")
	}
	return fmt.Sprintf("http://%s", addr)
}

// AutoRegisterAgent starts registry auto-registration when enabled by config.
func AutoRegisterAgent(ctx context.Context, cfg *config.Config, endpoint AgentEndpoint) (context.CancelFunc, error) {
	if cfg == nil {
		return nil, nil
	}
	entry := discovery.AgentEndpoint{
		Name:         endpoint.Name,
		AgentCardURL: endpoint.AgentCardURL,
		GRPCAddr:     endpoint.GRPCAddr,
		HTTPURL:      endpoint.HTTPURL,
		Labels:       map[string]string{"env": "demo"},
	}
	return discovery.StartAutoRegisterFromConfig(ctx, cfg, entry)
}
