package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// defaultOllamaTimeout is the http.Client timeout applied when the
// caller does not configure one explicitly. Local models can take
// well over a minute on a cold start, so the default is generous;
// callers expecting longer should pass WithOllamaTimeout (or 0 to
// fall back entirely to the request context's deadline).
const defaultOllamaTimeout = 120 * time.Second

// envOllamaTimeout names the environment variable parsed for the
// fallback http.Client timeout. The value is parsed with
// time.ParseDuration ("5m", "300s", "90s"). Set to "0" to disable
// the http.Client timeout and rely solely on the request context.
const envOllamaTimeout = "KAIROS_OLLAMA_TIMEOUT"

// OllamaProvider implements the Provider interface for Ollama.
type OllamaProvider struct {
	baseURL string
	client  *http.Client
}

// OllamaOption configures an OllamaProvider at construction time.
type OllamaOption func(*OllamaProvider)

// WithOllamaTimeout sets the underlying http.Client timeout. A zero
// duration disables the client-side timeout entirely so the request
// context is the only deadline that fires — appropriate for slow
// local models or callers that already wrap each Chat call with
// context.WithTimeout.
func WithOllamaTimeout(d time.Duration) OllamaOption {
	return func(p *OllamaProvider) {
		p.client.Timeout = d
	}
}

// WithOllamaHTTPClient replaces the http.Client outright. Use this
// when you need to inject custom transport, timeouts or middleware
// (proxies, instrumentation, retries) that the simple WithOllamaTimeout
// option cannot express.
func WithOllamaHTTPClient(c *http.Client) OllamaOption {
	return func(p *OllamaProvider) {
		if c != nil {
			p.client = c
		}
	}
}

// NewOllama creates a new OllamaProvider pointed at baseURL (defaults
// to http://localhost:11434 when empty). The http.Client timeout is
// resolved in the following order: explicit WithOllamaTimeout option
// > KAIROS_OLLAMA_TIMEOUT environment variable > defaultOllamaTimeout.
func NewOllama(baseURL string, opts ...OllamaOption) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	p := &OllamaProvider{
		baseURL: baseURL,
		client:  &http.Client{Timeout: ollamaTimeoutFromEnv()},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// ollamaTimeoutFromEnv reads KAIROS_OLLAMA_TIMEOUT and parses it as a
// Go duration. Falls back to defaultOllamaTimeout on any error so a
// typo in the env var doesn't lock callers out of all Ollama traffic.
func ollamaTimeoutFromEnv() time.Duration {
	raw := os.Getenv(envOllamaTimeout)
	if raw == "" {
		return defaultOllamaTimeout
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return defaultOllamaTimeout
	}
	return d
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
	bodyHandled := false
	defer func() {
		if !bodyHandled {
			resp.Body.Close()
		}
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama api returned status %d: %s", resp.StatusCode, string(respBody))
	}
	bodyHandled = true

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
