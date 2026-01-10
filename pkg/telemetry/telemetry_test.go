package telemetry

import (
	"context"
	"testing"
)

func TestInit(t *testing.T) {
	shutdown, err := Init("test-service", "v0.0.1")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Shutdown function should not be nil")
	}

	// Ensure shutdown works
	if err := shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}
}
