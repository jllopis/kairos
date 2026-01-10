package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Embedder implements the memory.Embedder interface using Ollama.
type Embedder struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewEmbedder creates a new Ollama Embedder.
func NewEmbedder(baseURL, model string) *Embedder {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &Embedder{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

type embeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// Embed converts a text string into a vector.
func (e *Embedder) Embed(ctx context.Context, text string) ([]float32, error) {
	req := embeddingRequest{
		Model:  e.model,
		Prompt: text,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama embedding api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama api returned status: %d", resp.StatusCode)
	}

	var embResp embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	// Convert float64 to float32
	vec := make([]float32, len(embResp.Embedding))
	for i, v := range embResp.Embedding {
		vec[i] = float32(v)
	}

	return vec, nil
}
