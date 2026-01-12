package discovery

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// AgentEndpoint represents a discovered agent entry.
type AgentEndpoint struct {
	Name         string
	AgentCardURL string
	GRPCAddr     string
	HTTPURL      string
	Labels       map[string]string
	ExpiresAt    time.Time
}

// Provider lists or registers agent endpoints.
type Provider interface {
	List(ctx context.Context) ([]AgentEndpoint, error)
}

// Resolver aggregates providers in priority order.
type Resolver struct {
	providers []Provider
}

// NewResolver creates a resolver with providers in order of priority.
func NewResolver(providers ...Provider) (*Resolver, error) {
	filtered := make([]Provider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		filtered = append(filtered, provider)
	}
	if len(filtered) == 0 {
		return nil, errors.New("no discovery providers configured")
	}
	return &Resolver{providers: filtered}, nil
}

// Resolve returns discovered endpoints in order, deduped by AgentCardURL.
func (r *Resolver) Resolve(ctx context.Context) ([]AgentEndpoint, error) {
	if r == nil {
		return nil, errors.New("resolver is nil")
	}
	out := make([]AgentEndpoint, 0)
	seen := map[string]struct{}{}
	for _, provider := range r.providers {
		entries, err := provider.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			key := normalizeKey(entry.AgentCardURL, entry.Name)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, entry)
		}
	}
	return out, nil
}

// ProviderOrder returns the configured provider order or defaults.
func ProviderOrder(order []string) []string {
	if len(order) == 0 {
		return []string{"config", "well_known", "registry"}
	}
	out := make([]string, 0, len(order))
	seen := map[string]struct{}{}
	for _, item := range order {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	if len(out) == 0 {
		return []string{"config", "well_known", "registry"}
	}
	return out
}

// Dedupe by AgentCardURL when possible, otherwise by name.
func normalizeKey(url, name string) string {
	url = strings.TrimSpace(strings.ToLower(url))
	if url != "" {
		return url
	}
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	return fmt.Sprintf("name:%s", name)
}

// SortByName sorts endpoints by name and then AgentCardURL.
func SortByName(endpoints []AgentEndpoint) {
	sort.Slice(endpoints, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(endpoints[i].Name))
		right := strings.ToLower(strings.TrimSpace(endpoints[j].Name))
		if left == right {
			return strings.ToLower(endpoints[i].AgentCardURL) < strings.ToLower(endpoints[j].AgentCardURL)
		}
		return left < right
	})
}
