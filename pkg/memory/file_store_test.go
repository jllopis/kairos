package memory

import (
	"context"
	"path/filepath"
	"testing"
)

func TestFileStoreRetrieveLast(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "memory.log"))

	if _, err := store.Retrieve(context.Background(), nil); err == nil {
		t.Fatal("expected ErrNotFound on empty store")
	}

	if err := store.Store(context.Background(), map[string]any{"n": 1}); err != nil {
		t.Fatalf("store failed: %v", err)
	}
	if err := store.Store(context.Background(), map[string]any{"n": 2}); err != nil {
		t.Fatalf("store failed: %v", err)
	}

	got, err := store.Retrieve(context.Background(), nil)
	if err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	data := got.(map[string]any)
	if data["n"] != float64(2) {
		t.Fatalf("expected 2, got %v", data["n"])
	}
}

func TestFileStoreRetrievePredicate(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "memory.log"))

	_ = store.Store(context.Background(), map[string]any{"label": "alpha"})
	_ = store.Store(context.Background(), map[string]any{"label": "beta"})

	got, err := store.Retrieve(context.Background(), func(v any) bool {
		entry, ok := v.(map[string]any)
		return ok && entry["label"] == "beta"
	})
	if err != nil {
		t.Fatalf("retrieve failed: %v", err)
	}
	entry := got.(map[string]any)
	if entry["label"] != "beta" {
		t.Fatalf("expected beta, got %v", entry["label"])
	}
}

func TestFileStoreUnsupportedQuery(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "memory.log"))
	_ = store.Store(context.Background(), "alpha")

	if _, err := store.Retrieve(context.Background(), 123); err == nil {
		t.Fatal("expected error for unsupported query type")
	}
}
