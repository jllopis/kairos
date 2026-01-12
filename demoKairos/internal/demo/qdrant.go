package demo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/memory/qdrant"
)

// QdrantConfig defines the connection settings for Qdrant.
type QdrantConfig struct {
	URL        string
	Collection string
}

// NewQdrantStore creates a Qdrant store client.
func NewQdrantStore(cfg QdrantConfig) (*qdrant.Store, error) {
	if cfg.URL == "" {
		cfg.URL = "localhost:6334"
	}
	return qdrant.New(cfg.URL)
}

// EnsureCollection ensures a collection exists with the embedder dimension.
func EnsureCollection(ctx context.Context, store memory.VectorStore, embedder memory.Embedder, name string) error {
	if name == "" {
		return fmt.Errorf("collection name is required")
	}
	vec, err := embedder.Embed(ctx, "dimension probe")
	if err != nil {
		return fmt.Errorf("embedder probe failed: %w", err)
	}
	if err := store.CreateCollection(ctx, name, uint64(len(vec))); err != nil {
		_, searchErr := store.Search(ctx, name, vec, 1, 0)
		if searchErr == nil {
			return nil
		}
		return err
	}
	return nil
}

// Doc represents a knowledge base entry.
type Doc struct {
	ID       string
	Text     string
	Metadata map[string]interface{}
}

// IngestDocs embeds and stores documents in Qdrant.
func IngestDocs(ctx context.Context, store memory.VectorStore, embedder memory.Embedder, collection string, docs []Doc) error {
	points := make([]memory.Point, 0, len(docs))
	for _, doc := range docs {
		if doc.Text == "" {
			continue
		}
		vec, err := embedder.Embed(ctx, doc.Text)
		if err != nil {
			return fmt.Errorf("embed doc: %w", err)
		}
		id := doc.ID
		if id == "" {
			id = uuid.NewString()
		}
		payload := map[string]interface{}{"text": doc.Text}
		for k, v := range doc.Metadata {
			payload[k] = v
		}
		points = append(points, memory.Point{
			ID:        id,
			Vector:    vec,
			Payload:   payload,
			Timestamp: time.Now().Unix(),
		})
	}
	if len(points) == 0 {
		return nil
	}
	return store.Upsert(ctx, collection, points)
}

// SearchDocs runs a semantic query and returns top results.
func SearchDocs(ctx context.Context, store memory.VectorStore, embedder memory.Embedder, collection, query string, limit int) ([]memory.SearchResult, error) {
	vec, err := embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if limit <= 0 {
		limit = 5
	}
	return store.Search(ctx, collection, vec, limit, 0.2)
}
