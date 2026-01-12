package discovery

import (
	"context"
	"strings"

	"github.com/jllopis/kairos/pkg/a2a/agentcard"
)

// WellKnownProvider discovers AgentCards from base URLs.
type WellKnownProvider struct {
	BaseURLs []string
}

// NewWellKnownProvider builds a provider for base URLs.
func NewWellKnownProvider(baseURLs []string) *WellKnownProvider {
	clean := make([]string, 0, len(baseURLs))
	seen := map[string]struct{}{}
	for _, url := range baseURLs {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		key := strings.ToLower(url)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		clean = append(clean, url)
	}
	return &WellKnownProvider{BaseURLs: clean}
}

// List fetches AgentCards and returns endpoints.
func (p *WellKnownProvider) List(ctx context.Context) ([]AgentEndpoint, error) {
	if p == nil || len(p.BaseURLs) == 0 {
		return nil, nil
	}
	out := make([]AgentEndpoint, 0, len(p.BaseURLs))
	for _, baseURL := range p.BaseURLs {
		card, err := agentcard.Fetch(ctx, baseURL)
		if err != nil {
			continue
		}
		out = append(out, AgentEndpoint{
			Name:         strings.TrimSpace(card.GetName()),
			AgentCardURL: strings.TrimRight(baseURL, "/") + agentcard.WellKnownPath,
		})
	}
	return out, nil
}
