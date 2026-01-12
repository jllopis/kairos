package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Client wraps the JSON-RPC binding for A2A.
type Client struct {
	endpoint   string
	httpClient *http.Client
	headers    map[string]string
}

// Option configures the client.
type Option func(*Client)

// New creates a JSON-RPC client bound to an HTTP endpoint.
func New(endpoint string, opts ...Option) *Client {
	client := &Client{
		endpoint:   endpoint,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

// WithHeaders sets default headers for each request.
func WithHeaders(headers map[string]string) Option {
	return func(c *Client) {
		c.headers = cloneHeaders(headers)
	}
}

// WithHTTPClient overrides the HTTP client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
		}
	}
}

// SendMessage invokes the SendMessage JSON-RPC method.
func (c *Client) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	resp := &a2av1.SendMessageResponse{}
	if err := c.call(ctx, "SendMessage", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SendStreamingMessage invokes SendStreamingMessage and streams responses via SSE.
func (c *Client) SendStreamingMessage(ctx context.Context, req *a2av1.SendMessageRequest) (<-chan *a2av1.StreamResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	return c.stream(ctx, "SendStreamingMessage", req)
}

// GetTask invokes the GetTask method.
func (c *Client) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	resp := &a2av1.Task{}
	if err := c.call(ctx, "GetTask", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ListTasks invokes the ListTasks method.
func (c *Client) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	resp := &a2av1.ListTasksResponse{}
	if err := c.call(ctx, "ListTasks", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelTask invokes the CancelTask method.
func (c *Client) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	resp := &a2av1.Task{}
	if err := c.call(ctx, "CancelTask", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SubscribeToTask streams task updates via SSE.
func (c *Client) SubscribeToTask(ctx context.Context, req *a2av1.SubscribeToTaskRequest) (<-chan *a2av1.StreamResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	return c.stream(ctx, "SubscribeToTask", req)
}

// GetExtendedAgentCard invokes GetExtendedAgentCard.
func (c *Client) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	if req == nil {
		req = &a2av1.GetExtendedAgentCardRequest{}
	}
	resp := &a2av1.AgentCard{}
	if err := c.call(ctx, "GetExtendedAgentCard", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetApproval retrieves a single approval record.
func (c *Client) GetApproval(ctx context.Context, id string) (*server.ApprovalRecord, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id is required")
	}
	var resp server.ApprovalRecord
	if err := c.callJSON(ctx, "GetApproval", map[string]string{"id": id}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListApprovals lists approvals using the filter.
func (c *Client) ListApprovals(ctx context.Context, filter server.ApprovalFilter) ([]*server.ApprovalRecord, error) {
	params := map[string]any{}
	if filter.TaskID != "" {
		params["task_id"] = filter.TaskID
	}
	if filter.ContextID != "" {
		params["context_id"] = filter.ContextID
	}
	if filter.Status != "" {
		params["status"] = string(filter.Status)
	}
	if filter.Limit > 0 {
		params["limit"] = filter.Limit
	}
	if !filter.ExpiringBefore.IsZero() {
		params["expires_before"] = filter.ExpiringBefore.UnixMilli()
	}
	var resp []*server.ApprovalRecord
	if err := c.callJSON(ctx, "ListApprovals", params, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ApproveApproval approves a pending approval and executes the task.
func (c *Client) ApproveApproval(ctx context.Context, id, reason string) (*a2av1.Task, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id is required")
	}
	resp := &a2av1.Task{}
	params := map[string]string{"id": id, "reason": reason}
	if err := c.callJSON(ctx, "ApproveApproval", params, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// RejectApproval rejects a pending approval.
func (c *Client) RejectApproval(ctx context.Context, id, reason string) (*a2av1.Task, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id is required")
	}
	resp := &a2av1.Task{}
	params := map[string]string{"id": id, "reason": reason}
	if err := c.callJSON(ctx, "RejectApproval", params, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) call(ctx context.Context, method string, params proto.Message, result proto.Message) error {
	payload, err := protojson.Marshal(params)
	if err != nil {
		return err
	}
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  method,
		Params:  json.RawMessage(payload),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	c.applyHeaders(ctx, request)
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseHTTPError(resp)
	}
	var decoded rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return err
	}
	if decoded.Error != nil {
		return status.Error(codes.Unknown, decoded.Error.Message)
	}
	if result == nil {
		return nil
	}
	if err := protojson.Unmarshal(decoded.Result, result); err != nil {
		return err
	}
	return nil
}

func (c *Client) callJSON(ctx context.Context, method string, params any, result any) error {
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  method,
		Params:  json.RawMessage(payload),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	c.applyHeaders(ctx, request)
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseHTTPError(resp)
	}
	var decoded rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return err
	}
	if decoded.Error != nil {
		return status.Error(codes.Unknown, decoded.Error.Message)
	}
	if result == nil {
		return nil
	}
	if msg, ok := result.(proto.Message); ok {
		return protojson.Unmarshal(decoded.Result, msg)
	}
	return json.Unmarshal(decoded.Result, result)
}

func (c *Client) stream(ctx context.Context, method string, params proto.Message) (<-chan *a2av1.StreamResponse, error) {
	payload, err := protojson.Marshal(params)
	if err != nil {
		return nil, err
	}
	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      uuid.NewString(),
		Method:  method,
		Params:  json.RawMessage(payload),
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	c.applyHeaders(ctx, request)
	resp, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, parseHTTPError(resp)
	}
	out := make(chan *a2av1.StreamResponse)
	go func() {
		defer resp.Body.Close()
		defer close(out)
		_ = readSSE(ctx, resp.Body, func(payload []byte) error {
			var decoded rpcResponse
			if err := json.Unmarshal(payload, &decoded); err != nil {
				return err
			}
			if decoded.Error != nil {
				return status.Error(codes.Unknown, decoded.Error.Message)
			}
			streamResp := &a2av1.StreamResponse{}
			if err := protojson.Unmarshal(decoded.Result, streamResp); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- streamResp:
				return nil
			}
		})
	}()
	return out, nil
}

func (c *Client) applyHeaders(ctx context.Context, request *http.Request) {
	for key, value := range c.headers {
		request.Header.Set(key, value)
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(request.Header))
}

func readSSE(ctx context.Context, body io.Reader, handle func([]byte) error) error {
	reader := bufio.NewReader(body)
	var buffer bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				if buffer.Len() > 0 {
					_ = handle(buffer.Bytes())
				}
				return nil
			}
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			if buffer.Len() == 0 {
				continue
			}
			if err := handle(buffer.Bytes()); err != nil {
				return err
			}
			buffer.Reset()
			continue
		}
		if strings.HasPrefix(line, "data:") {
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if buffer.Len() > 0 {
				buffer.WriteByte('\n')
			}
			buffer.WriteString(payload)
		}
	}
}

func parseHTTPError(response *http.Response) error {
	payload, _ := io.ReadAll(response.Body)
	if len(payload) == 0 {
		return status.Error(codes.Unknown, response.Status)
	}
	var decoded struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return status.Error(codes.Unknown, response.Status)
	}
	detail := strings.TrimSpace(decoded.Detail)
	if detail == "" {
		detail = strings.TrimSpace(decoded.Title)
	}
	if detail == "" {
		detail = response.Status
	}
	return status.Error(codes.Unknown, detail)
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func cloneHeaders(headers map[string]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		out[key] = value
	}
	return out
}
