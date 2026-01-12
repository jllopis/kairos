package discovery

import (
	"context"
	"testing"
)

type staticProvider struct {
	entries []AgentEndpoint
	fail    error
}

func (p staticProvider) List(_ context.Context) ([]AgentEndpoint, error) {
	if p.fail != nil {
		return nil, p.fail
	}
	return p.entries, nil
}

func TestResolverResolveOrderAndDedupe(t *testing.T) {
	providers := []Provider{
		staticProvider{entries: []AgentEndpoint{{Name: "alpha", AgentCardURL: "http://a"}}},
		staticProvider{entries: []AgentEndpoint{{Name: "alpha", AgentCardURL: "http://a"}, {Name: "beta", AgentCardURL: "http://b"}}},
	}
	resolver, err := NewResolver(providers...)
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	entries, err := resolver.Resolve(context.Background())
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].AgentCardURL != "http://a" || entries[1].AgentCardURL != "http://b" {
		t.Fatalf("unexpected order: %+v", entries)
	}
}

func TestProviderOrderDefaults(t *testing.T) {
	order := ProviderOrder(nil)
	if len(order) != 3 || order[0] != "config" || order[1] != "well_known" || order[2] != "registry" {
		t.Fatalf("unexpected default order: %v", order)
	}
}

func TestProviderOrderNormalization(t *testing.T) {
	order := ProviderOrder([]string{"Config", "", "well_known", "config", "registry"})
	if len(order) != 3 {
		t.Fatalf("unexpected order length: %v", order)
	}
	if order[0] != "config" || order[1] != "well_known" || order[2] != "registry" {
		t.Fatalf("unexpected order: %v", order)
	}
}
