package memory

import (
	"context"
	"testing"
)

func TestInMemoryRetrieveLast(t *testing.T) {
	store := NewInMemory()
	if _, err := store.Retrieve(context.Background(), nil); err == nil {
		t.Fatal("expected ErrNotFound on empty store")
	}

	if err := store.Store(context.Background(), "first"); err != nil {
		t.Fatalf("store failed: %v", err)
	}
	if err := store.Store(context.Background(), "second"); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	got, err := store.Retrieve(context.Background(), nil)
	if err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	if got != "second" {
		t.Fatalf("expected second, got %v", got)
	}
}

func TestInMemoryRetrievePredicate(t *testing.T) {
	store := NewInMemory()
	_ = store.Store(context.Background(), "alpha")
	_ = store.Store(context.Background(), "beta")
	_ = store.Store(context.Background(), "gamma")

	got, err := store.Retrieve(context.Background(), func(v any) bool {
		return v == "beta"
	})
	if err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	if got != "beta" {
		t.Fatalf("expected beta, got %v", got)
	}
}

func TestInMemoryUnsupportedQuery(t *testing.T) {
	store := NewInMemory()
	_ = store.Store(context.Background(), "alpha")

	if _, err := store.Retrieve(context.Background(), 42); err == nil {
		t.Fatal("expected error for unsupported query type")
	}
}
