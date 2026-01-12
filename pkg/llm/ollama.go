package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaProvider implements the Provider interface for Ollama.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// NewOllama creates a new OllamaProvider.
func NewOllama(baseURL string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

type ollamaRequest struct {
	Model    string                 `json:"model"`
	Messages []Message              `json:"messages"`
	Stream   bool                   `json:"stream"`
	Tools    []Tool                 `json:"tools,omitempty"`
	Options  map[string]interface{} `json:"options,omitempty"`
}

type ollamaResponse struct {
	Message         Message `json:"message"`
	Done            bool    `json:"done"`
	TotalDuration   int64   `json:"total_duration"` // nanos
	EvalCount       int     `json:"eval_count"`
	PromptEvalCount int     `json:"prompt_eval_count"`
}

// Chat sends a chat request to Ollama and maps the response to ChatResponse.
func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	oReq := ollamaRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
		Tools:    req.Tools,
	}

	if req.Temperature != 0 {
		oReq.Options = map[string]interface{}{
			"temperature": req.Temperature,
		}
	}

	body, err := json.Marshal(oReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama api call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama api returned status: %d", resp.StatusCode)
	}

	var oResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&oResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return &ChatResponse{
		Content:   oResp.Message.Content,
		ToolCalls: oResp.Message.ToolCalls,
		Usage: Usage{
			PromptTokens:     oResp.PromptEvalCount,
			CompletionTokens: oResp.EvalCount,
			TotalTokens:      oResp.PromptEvalCount + oResp.EvalCount,
		},
	}, nil
}
