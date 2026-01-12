package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RegistryProvider queries an external registry service.
type RegistryProvider struct {
	BaseURL   string
	HTTP      *http.Client
	AuthToken string
}

// NewRegistryProvider creates a registry provider pointing at baseURL.
func NewRegistryProvider(baseURL string) *RegistryProvider {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	return &RegistryProvider{
		BaseURL: baseURL,
		HTTP:    http.DefaultClient,
	}
}

// List returns active endpoints from the registry.
func (p *RegistryProvider) List(ctx context.Context) ([]AgentEndpoint, error) {
	if p == nil || p.BaseURL == "" {
		return nil, nil
	}
	url := p.BaseURL + "/v1/agents"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(p.AuthToken) != "" {
		req.Header.Set("Authorization", "Bearer "+p.AuthToken)
	}
	resp, err := p.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry list failed: %s", resp.Status)
	}
	var out []AgentEndpoint
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// Register registers an endpoint in the registry.
func (p *RegistryProvider) Register(ctx context.Context, endpoint AgentEndpoint) error {
	if p == nil || p.BaseURL == "" {
		return errors.New("registry base url not configured")
	}
	url := p.BaseURL + "/v1/agents"
	payload, err := json.Marshal(endpoint)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(p.AuthToken) != "" {
		req.Header.Set("Authorization", "Bearer "+p.AuthToken)
	}
	resp, err := p.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry register failed: %s", resp.Status)
	}
	return nil
}

func (p *RegistryProvider) http() *http.Client {
	if p != nil && p.HTTP != nil {
		return p.HTTP
	}
	return http.DefaultClient
}

// RegistryServer is a minimal in-process registry (optional helper).
type RegistryServer struct {
	Addr       string
	TTL        time.Duration
	endpoints  map[string]AgentEndpoint
	lastUpdate map[string]time.Time
}

// NewRegistryServer builds a registry server with TTL.
func NewRegistryServer(addr string, ttl time.Duration) *RegistryServer {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &RegistryServer{
		Addr:       addr,
		TTL:        ttl,
		endpoints:  map[string]AgentEndpoint{},
		lastUpdate: map[string]time.Time{},
	}
}

// Handler returns the HTTP handler for the registry API.
func (r *RegistryServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/agents", r.handleAgents)
	return mux
}

// Serve starts the registry HTTP server.
func (r *RegistryServer) Serve() error {
	if r == nil {
		return errors.New("registry server is nil")
	}
	return http.ListenAndServe(r.Addr, r.Handler())
}

func (r *RegistryServer) handleAgents(w http.ResponseWriter, req *http.Request) {
	if r == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	switch req.Method {
	case http.MethodGet:
		r.handleList(w)
	case http.MethodPost:
		r.handleRegister(w, req)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (r *RegistryServer) handleList(w http.ResponseWriter) {
	now := time.Now().UTC()
	out := make([]AgentEndpoint, 0, len(r.endpoints))
	for key, entry := range r.endpoints {
		last := r.lastUpdate[key]
		if now.Sub(last) > r.TTL {
			delete(r.endpoints, key)
			delete(r.lastUpdate, key)
			continue
		}
		out = append(out, entry)
	}
	payload, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(payload)
}

func (r *RegistryServer) handleRegister(w http.ResponseWriter, req *http.Request) {
	var endpoint AgentEndpoint
	if err := json.NewDecoder(req.Body).Decode(&endpoint); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	key := normalizeKey(endpoint.AgentCardURL, endpoint.Name)
	if key == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	endpoint.ExpiresAt = time.Now().UTC().Add(r.TTL)
	r.endpoints[key] = endpoint
	r.lastUpdate[key] = time.Now().UTC()
	w.WriteHeader(http.StatusNoContent)
}
