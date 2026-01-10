package memory

import "context"

// VectorStore defines the interface for a vector database.
type VectorStore interface {
	// Upsert adds or updates points in the vector store.
	Upsert(ctx context.Context, collection string, points []Point) error
	// Search searches for the nearest vectors to the given vector.
	Search(ctx context.Context, collection string, vector []float32, limit int, scoreThreshold float32) ([]SearchResult, error)
	// CreateCollection creates a new collection if it doesn't exist.
	CreateCollection(ctx context.Context, name string, vectorSize uint64) error
}

// Point represents a data point in the vector store.
type Point struct {
	ID        string                 `json:"id"`
	Vector    []float32              `json:"vector"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp int64                  `json:"timestamp"`
}

// SearchResult represents a result from a vector search.
type SearchResult struct {
	ID    string  `json:"id"`
	Score float32 `json:"score"`
	Point Point   `json:"point"`
}

// Embedder defines the interface for converting text to vectors.
type Embedder interface {
	// Embed converts a text string into a vector.
	Embed(ctx context.Context, text string) ([]float32, error)
}
