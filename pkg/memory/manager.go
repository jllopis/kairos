package memory

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// VectorMemory implements the core.Memory interface using a vector store and embedder.
type VectorMemory struct {
	store      VectorStore
	embedder   Embedder
	collection string
}

// NewVectorMemory creates a new VectorMemory instance.
func NewVectorMemory(ctx context.Context, store VectorStore, embedder Embedder, collection string) (*VectorMemory, error) {
	// Ensure collection exists
	// For Qdrant, typically 1536 for OpenAI, 768 for many OSS models, 1024 for others.
	// We need to know the dimension.
	// For now, let's assume the user has created the collection OR we try to embed a dummy to get dim.
	// Or we just try to create it with a default size if we know it.
	// Ideally, pass dimension in config.

	// For this implementation, we will expect the collection to be created or we lazily create it on first storage if possible.
	// However, CreateCollection requires dimension.

	// Let's just store the params.
	return &VectorMemory{
		store:      store,
		embedder:   embedder,
		collection: collection,
	}, nil
}

// Initialize ensures the collection exists with the correct dimension.
func (vm *VectorMemory) Initialize(ctx context.Context) error {
	// Embed a sample text to get dimensions
	vec, err := vm.embedder.Embed(ctx, "hello")
	if err != nil {
		return fmt.Errorf("failed to get embedding dimension: %w", err)
	}

	// Try to create collection. Ignore error if it already exists (store implementation should handle this check or error).
	// Our defined interface `CreateCollection` returns error. We should check if it's "already exists".
	// For now, let's just attempt and log/ignore if specific error, or rely on store to be idempotent-ish or checked.
	// Since our interface is simple, let's assume we can call CreateCollection and if it fails, we assume it exists for now.
	// A better approach is to add Exists or List methods to VectorStore.

	// For now, simply calling CreateCollection. Qdrant returns error if exists.
	// We should probably check if it exists first?
	// Let's proceed.

	err = vm.store.CreateCollection(ctx, vm.collection, uint64(len(vec)))
	if err != nil {
		// In a real app we'd check if "already exists" error.
		// For this demo, let's assume if it fails it might be because it exists.
		// We will just return nil to be safe, or log it.
		// Returing the error might block startup if it exists.
		// Let's assume the store implementation handles "if not exists" logic or we tolerate failure.
		// The Qdrant implementation returns error if API fails.
		// We will return the error for now, developer should handle it or we add more robust check later.
		// Actually, let's treat it as non-fatal if "already exists" but we can't distinguish easily without parsing error string which is brittle.
		// Plan: Try to search. If search works, collection exists.
		_, searchErr := vm.store.Search(ctx, vm.collection, vec, 1, 0.0)
		if searchErr == nil {
			return nil // Collection exists
		}
		return err // Return the creation error
	}
	return nil
}

// Store saves data into the vector memory.
// Expects data to be a string or a struct with "Text" field.
func (vm *VectorMemory) Store(ctx context.Context, data any) error {
	text, ok := data.(string)
	if !ok {
		return fmt.Errorf("VectorMemory currently only supports string data")
	}

	vector, err := vm.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("failed to embed text: %w", err)
	}

	id := uuid.New().String()
	point := Point{
		ID:     id,
		Vector: vector,
		Payload: map[string]interface{}{
			"text":      text,
			"timestamp": time.Now().Unix(),
		},
		Timestamp: time.Now().Unix(),
	}

	if err := vm.store.Upsert(ctx, vm.collection, []Point{point}); err != nil {
		return fmt.Errorf("failed to store point: %w", err)
	}

	return nil
}

// Retrieve finds relevant data in the vector memory.
// Expects query to be a string.
func (vm *VectorMemory) Retrieve(ctx context.Context, query any) (any, error) {
	text, ok := query.(string)
	if !ok {
		return nil, fmt.Errorf("VectorMemory currently only supports string queries")
	}

	vector, err := vm.embedder.Embed(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Limit 5, threshold can be tuneable.
	results, err := vm.store.Search(ctx, vm.collection, vector, 5, 0.6)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Return list of strings
	var matches []string
	for _, r := range results {
		if val, ok := r.Point.Payload["text"].(string); ok {
			matches = append(matches, val)
		}
	}

	return matches, nil
}
