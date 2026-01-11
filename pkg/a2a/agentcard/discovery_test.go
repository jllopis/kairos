package agentcard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPublishHandler_NoCard(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	rec := httptest.NewRecorder()

	PublishHandler(nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestPublishHandler_ServesCard(t *testing.T) {
	card := &a2av1.AgentCard{
		ProtocolVersion: strPtr("1.0"),
		Name:            "demo-agent",
	}
	req := httptest.NewRequest(http.MethodGet, WellKnownPath, nil)
	rec := httptest.NewRecorder()

	PublishHandler(card).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != DefaultMediaType {
		t.Fatalf("expected content type %q", DefaultMediaType)
	}
	if rec.Body.Len() == 0 {
		t.Fatalf("expected non-empty body")
	}
}

func TestFetch_Success(t *testing.T) {
	card := &a2av1.AgentCard{
		ProtocolVersion: strPtr("1.0"),
		Name:            "demo-agent",
	}
	payload, err := protojson.Marshal(card)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != DefaultMediaType {
			http.Error(w, "invalid accept", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", DefaultMediaType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	got, err := Fetch(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if got.GetName() != "demo-agent" {
		t.Fatalf("expected name %q, got %q", "demo-agent", got.GetName())
	}
}

func TestFetch_NonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	if _, err := Fetch(context.Background(), server.URL); err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

func strPtr(value string) *string {
	return &value
}
