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
	"net/url"
	"strings"

	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// Client wraps the HTTP+JSON binding for A2A.
type Client struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// Option configures the client.
type Option func(*Client)

// New creates a new HTTP+JSON A2A client.
func New(baseURL string, opts ...Option) *Client {
	client := &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
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

// SendMessage calls the message:send endpoint.
func (c *Client) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	resp := &a2av1.SendMessageResponse{}
	if err := c.doProto(ctx, http.MethodPost, "/message:send", req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SendStreamingMessage calls message:stream and returns a stream of responses.
func (c *Client) SendStreamingMessage(ctx context.Context, req *a2av1.SendMessageRequest) (<-chan *a2av1.StreamResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	return c.streamProto(ctx, http.MethodPost, "/message:stream", req, func(payload []byte) (*a2av1.StreamResponse, error) {
		resp := &a2av1.StreamResponse{}
		if err := protojson.Unmarshal(payload, resp); err != nil {
			return nil, err
		}
		return resp, nil
	})
}

// GetTask retrieves a task by name.
func (c *Client) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	path, err := taskPath(req.GetName())
	if err != nil {
		return nil, err
	}
	query := url.Values{}
	if req.HistoryLength != nil {
		query.Set("historyLength", fmt.Sprintf("%d", req.GetHistoryLength()))
	}
	endpoint := withQuery(c.endpoint(path), query)
	resp := &a2av1.Task{}
	if err := c.doProto(ctx, http.MethodGet, endpoint, nil, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ListTasks lists tasks based on request filters.
func (c *Client) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	query := url.Values{}
	if req.GetContextId() != "" {
		query.Set("contextId", req.GetContextId())
	}
	if req.GetPageToken() != "" {
		query.Set("pageToken", req.GetPageToken())
	}
	if req.GetLastUpdatedAfter() > 0 {
		query.Set("lastUpdatedAfter", fmt.Sprintf("%d", req.GetLastUpdatedAfter()))
	}
	if req.GetStatus() != a2av1.TaskState_TASK_STATE_UNSPECIFIED {
		query.Set("status", req.GetStatus().String())
	}
	if req.PageSize != nil {
		query.Set("pageSize", fmt.Sprintf("%d", req.GetPageSize()))
	}
	if req.HistoryLength != nil {
		query.Set("historyLength", fmt.Sprintf("%d", req.GetHistoryLength()))
	}
	if req.IncludeArtifacts != nil {
		query.Set("includeArtifacts", fmt.Sprintf("%t", req.GetIncludeArtifacts()))
	}
	endpoint := withQuery(c.endpoint("/tasks"), query)
	resp := &a2av1.ListTasksResponse{}
	if err := c.doProto(ctx, http.MethodGet, endpoint, nil, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelTask cancels a task.
func (c *Client) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	path, err := taskPath(req.GetName())
	if err != nil {
		return nil, err
	}
	endpoint := c.endpoint(path + ":cancel")
	resp := &a2av1.Task{}
	if err := c.doProto(ctx, http.MethodPost, endpoint, req, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// SubscribeToTask subscribes to task updates via SSE.
func (c *Client) SubscribeToTask(ctx context.Context, req *a2av1.SubscribeToTaskRequest) (<-chan *a2av1.StreamResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	path, err := taskPath(req.GetName())
	if err != nil {
		return nil, err
	}
	endpoint := c.endpoint(path + ":subscribe")
	return c.streamProto(ctx, http.MethodGet, endpoint, nil, func(payload []byte) (*a2av1.StreamResponse, error) {
		resp := &a2av1.StreamResponse{}
		if err := protojson.Unmarshal(payload, resp); err != nil {
			return nil, err
		}
		return resp, nil
	})
}

// GetExtendedAgentCard requests the extended agent card.
func (c *Client) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	resp := &a2av1.AgentCard{}
	if err := c.doProto(ctx, http.MethodGet, "/extendedAgentCard", nil, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetApproval retrieves a single approval record.
func (c *Client) GetApproval(ctx context.Context, id string) (*server.ApprovalRecord, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id is required")
	}
	endpoint := c.endpoint("/approvals/" + id)
	var resp server.ApprovalRecord
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListApprovals lists approvals using the filter.
func (c *Client) ListApprovals(ctx context.Context, filter server.ApprovalFilter) ([]*server.ApprovalRecord, error) {
	query := url.Values{}
	if filter.TaskID != "" {
		query.Set("taskId", filter.TaskID)
	}
	if filter.ContextID != "" {
		query.Set("contextId", filter.ContextID)
	}
	if filter.Status != "" {
		query.Set("status", string(filter.Status))
	}
	if filter.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", filter.Limit))
	}
	if !filter.ExpiringBefore.IsZero() {
		query.Set("expiresBefore", fmt.Sprintf("%d", filter.ExpiringBefore.UnixMilli()))
	}
	endpoint := withQuery(c.endpoint("/approvals"), query)
	var resp []*server.ApprovalRecord
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// ApproveApproval approves a pending request and executes the task.
func (c *Client) ApproveApproval(ctx context.Context, id, reason string) (*a2av1.Task, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id is required")
	}
	payload := map[string]string{"reason": reason}
	resp := &a2av1.Task{}
	if err := c.doJSON(ctx, http.MethodPost, c.endpoint("/approvals/"+id+":approve"), payload, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// RejectApproval rejects a pending request.
func (c *Client) RejectApproval(ctx context.Context, id, reason string) (*a2av1.Task, error) {
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("id is required")
	}
	payload := map[string]string{"reason": reason}
	resp := &a2av1.Task{}
	if err := c.doJSON(ctx, http.MethodPost, c.endpoint("/approvals/"+id+":reject"), payload, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *Client) endpoint(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return c.baseURL + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) doProto(ctx context.Context, method, endpoint string, req proto.Message, resp proto.Message) error {
	var body io.Reader
	if req != nil {
		payload, err := protojson.Marshal(req)
		if err != nil {
			return err
		}
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.endpoint(endpoint), body)
	if err != nil {
		return err
	}
	if req != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	c.applyHeaders(ctx, request)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return parseHTTPError(response)
	}
	if resp == nil {
		return nil
	}
	payload, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := protojson.Unmarshal(payload, resp); err != nil {
		return err
	}
	return nil
}

func (c *Client) doJSON(ctx context.Context, method, endpoint string, payload any, resp any) error {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(raw)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.endpoint(endpoint), body)
	if err != nil {
		return err
	}
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	c.applyHeaders(ctx, request)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return parseHTTPError(response)
	}
	if resp == nil {
		return nil
	}
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if msg, ok := resp.(proto.Message); ok {
		return protojson.Unmarshal(bodyBytes, msg)
	}
	return json.Unmarshal(bodyBytes, resp)
}

func (c *Client) streamProto(ctx context.Context, method, endpoint string, req proto.Message, parse func([]byte) (*a2av1.StreamResponse, error)) (<-chan *a2av1.StreamResponse, error) {
	var body io.Reader
	if req != nil && method != http.MethodGet {
		payload, err := protojson.Marshal(req)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.endpoint(endpoint), body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "text/event-stream")
	if req != nil && method != http.MethodGet {
		request.Header.Set("Content-Type", "application/json")
	}
	c.applyHeaders(ctx, request)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		defer response.Body.Close()
		return nil, parseHTTPError(response)
	}
	out := make(chan *a2av1.StreamResponse)
	go func() {
		defer response.Body.Close()
		defer close(out)
		_ = readSSE(ctx, response.Body, func(payload []byte) error {
			resp, err := parse(payload)
			if err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case out <- resp:
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

func taskPath(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", fmt.Errorf("task name is required")
	}
	if strings.HasPrefix(trimmed, "tasks/") {
		return "/" + trimmed, nil
	}
	return "/tasks/" + trimmed, nil
}

func withQuery(endpoint string, query url.Values) string {
	if len(query) == 0 {
		return endpoint
	}
	return endpoint + "?" + query.Encode()
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
