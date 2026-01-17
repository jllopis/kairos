package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// ChatStream implements StreamingProvider for streaming responses.
func (p *OllamaProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	oReq := ollamaRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true, // Enable streaming
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

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama api returned status %d: %s", resp.StatusCode, string(respBody))
	}

	// Create output channel
	chunks := make(chan StreamChunk, 100)

	// Process NDJSON stream in goroutine
	go func() {
		defer close(chunks)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		var accumulatedToolCalls []ToolCall
		var totalUsage Usage

		for {
			select {
			case <-ctx.Done():
				chunks <- StreamChunk{Error: ctx.Err()}
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					chunks <- StreamChunk{Error: err}
				}
				return
			}

			// Parse NDJSON line
			var event ollamaStreamEvent
			if err := json.Unmarshal(line, &event); err != nil {
				continue // Skip malformed lines
			}

			chunk := StreamChunk{}

			// Content from message
			if event.Message.Content != "" {
				chunk.Content = event.Message.Content
			}

			// Tool calls (Ollama sends complete tool calls, not deltas)
			if len(event.Message.ToolCalls) > 0 {
				accumulatedToolCalls = event.Message.ToolCalls
			}

			// Check if stream is done
			if event.Done {
				totalUsage = Usage{
					PromptTokens:     event.PromptEvalCount,
					CompletionTokens: event.EvalCount,
					TotalTokens:      event.PromptEvalCount + event.EvalCount,
				}
				chunks <- StreamChunk{
					Done:      true,
					ToolCalls: accumulatedToolCalls,
					Usage:     &totalUsage,
				}
				return
			}

			// Send chunk if there's content
			if chunk.Content != "" {
				chunks <- chunk
			}
		}
	}()

	return chunks, nil
}

// ollamaStreamEvent represents a streaming response from Ollama (NDJSON format).
type ollamaStreamEvent struct {
	Model           string  `json:"model"`
	CreatedAt       string  `json:"created_at"`
	Message         Message `json:"message"`
	Done            bool    `json:"done"`
	TotalDuration   int64   `json:"total_duration,omitempty"`
	LoadDuration    int64   `json:"load_duration,omitempty"`
	PromptEvalCount int     `json:"prompt_eval_count,omitempty"`
	EvalCount       int     `json:"eval_count,omitempty"`
	EvalDuration    int64   `json:"eval_duration,omitempty"`
}

// Ensure OllamaProvider implements StreamingProvider.
var _ StreamingProvider = (*OllamaProvider)(nil)
