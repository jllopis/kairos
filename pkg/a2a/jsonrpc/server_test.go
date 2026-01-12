package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

type testHandler struct {
	sendMessage   func(context.Context, *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error)
	sendStreaming func(*a2av1.SendMessageRequest, a2av1.A2AService_SendStreamingMessageServer) error
	getTask       func(context.Context, *a2av1.GetTaskRequest) (*a2av1.Task, error)
	listTasks     func(context.Context, *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error)
	cancelTask    func(context.Context, *a2av1.CancelTaskRequest) (*a2av1.Task, error)
	subscribe     func(*a2av1.SubscribeToTaskRequest, a2av1.A2AService_SubscribeToTaskServer) error
	card          func(context.Context, *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error)
}

func (h *testHandler) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	if h.sendMessage != nil {
		return h.sendMessage(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "SendMessage not configured")
}

func (h *testHandler) SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
	if h.sendStreaming != nil {
		return h.sendStreaming(req, stream)
	}
	return status.Error(codes.Unimplemented, "SendStreamingMessage not configured")
}

func (h *testHandler) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	if h.getTask != nil {
		return h.getTask(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "GetTask not configured")
}

func (h *testHandler) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	if h.listTasks != nil {
		return h.listTasks(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "ListTasks not configured")
}

func (h *testHandler) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	if h.cancelTask != nil {
		return h.cancelTask(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "CancelTask not configured")
}

func (h *testHandler) SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error {
	if h.subscribe != nil {
		return h.subscribe(req, stream)
	}
	return status.Error(codes.Unimplemented, "SubscribeToTask not configured")
}

func (h *testHandler) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	if h.card != nil {
		return h.card(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "GetExtendedAgentCard not configured")
}

type streamRecorder struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{header: make(http.Header)}
}

func (r *streamRecorder) Header() http.Header {
	return r.header
}

func (r *streamRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(data)
}

func (r *streamRecorder) WriteHeader(status int) {
	r.status = status
}

func (r *streamRecorder) Flush() {}

func TestServerSendMessage(t *testing.T) {
	handler := &testHandler{
		sendMessage: func(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
			msg := &a2av1.Message{Role: a2av1.Role_ROLE_AGENT, Parts: []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ok"}}}}
			return &a2av1.SendMessageResponse{Payload: &a2av1.SendMessageResponse_Msg{Msg: msg}}, nil
		},
	}
	srv := New(handler)
	params, err := protojson.Marshal(&a2av1.SendMessageRequest{
		Request: &a2av1.Message{Role: a2av1.Role_ROLE_USER, Parts: []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ping"}}}},
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "SendMessage",
		Params:  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	raw, ok := payload["result"]
	if !ok {
		t.Fatalf("missing result")
	}
	var resp a2av1.SendMessageResponse
	if err := protojson.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if got := resp.GetMsg().GetParts()[0].GetText(); got != "ok" {
		t.Fatalf("expected ok, got %q", got)
	}
}

func TestServerMethodNotFound(t *testing.T) {
	srv := New(&testHandler{})
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      "1",
		Method:  "Unknown",
		Params:  json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if _, ok := payload["error"]; !ok {
		t.Fatalf("expected error payload")
	}
}

func TestServerSendStreamingMessage(t *testing.T) {
	handler := &testHandler{
		sendStreaming: func(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
			return stream.Send(&a2av1.StreamResponse{
				Payload: &a2av1.StreamResponse_StatusUpdate{
					StatusUpdate: &a2av1.TaskStatusUpdateEvent{
						Status: &a2av1.TaskStatus{State: a2av1.TaskState_TASK_STATE_WORKING},
					},
				},
			})
		},
	}
	srv := New(handler)
	params, err := protojson.Marshal(&a2av1.SendMessageRequest{
		Request: &a2av1.Message{Role: a2av1.Role_ROLE_USER, Parts: []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ping"}}}},
	})
	if err != nil {
		t.Fatalf("marshal params: %v", err)
	}
	reqBody, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		ID:      "stream-1",
		Method:  "SendStreamingMessage",
		Params:  params,
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(reqBody))
	rec := newStreamRecorder()
	srv.ServeHTTP(rec, req)
	if rec.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.status)
	}
	parts := strings.Split(strings.TrimSpace(rec.body.String()), "\n\n")
	if len(parts) == 0 || !strings.HasPrefix(parts[0], "data: ") {
		t.Fatalf("expected stream data, got %q", rec.body.String())
	}
	data := strings.TrimPrefix(parts[0], "data: ")
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	raw, ok := payload["result"]
	if !ok {
		t.Fatalf("missing result")
	}
	var resp a2av1.StreamResponse
	if err := protojson.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if resp.GetStatusUpdate() == nil {
		t.Fatalf("expected status update")
	}
}
