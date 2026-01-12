package agentcard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/encoding/protojson"
)

// Discovery constants for AgentCard HTTP endpoints.
const (
	// WellKnownPath is the standardized location for AgentCard discovery.
	WellKnownPath = "/.well-known/agent-card.json"
	// DefaultMediaType is the A2A media type for JSON payloads.
	DefaultMediaType = "application/a2a+json"
)

// PublishHandler serves the provided AgentCard as JSON.
func PublishHandler(card *a2av1.AgentCard) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if card == nil {
			http.Error(w, "agent card not configured", http.StatusNotFound)
			return
		}
		payload, err := protojson.Marshal(card)
		if err != nil {
			http.Error(w, "failed to encode agent card", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", DefaultMediaType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	})
}

// Fetch retrieves an AgentCard from a base URL.
func Fetch(ctx context.Context, baseURL string) (*a2av1.AgentCard, error) {
	url := strings.TrimRight(baseURL, "/") + WellKnownPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", DefaultMediaType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agent card fetch failed: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var card a2av1.AgentCard
	if err := protojson.Unmarshal(body, &card); err == nil {
		return &card, nil
	}

	if err := json.Unmarshal(body, &card); err != nil {
		return nil, err
	}
	return &card, nil
}
