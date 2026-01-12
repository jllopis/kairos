package httpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jllopis/kairos/pkg/a2a/server"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
)

type testHandler struct {
	sendMessage      func(context.Context, *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error)
	sendStreaming    func(*a2av1.SendMessageRequest, a2av1.A2AService_SendStreamingMessageServer) error
	getTask          func(context.Context, *a2av1.GetTaskRequest) (*a2av1.Task, error)
	listTasks        func(context.Context, *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error)
	cancelTask       func(context.Context, *a2av1.CancelTaskRequest) (*a2av1.Task, error)
	subscribeToTask  func(*a2av1.SubscribeToTaskRequest, a2av1.A2AService_SubscribeToTaskServer) error
	getExtendedCard  func(context.Context, *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error)
	setPushConfig    func(context.Context, *a2av1.SetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error)
	getPushConfig    func(context.Context, *a2av1.GetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error)
	listPushConfig   func(context.Context, *a2av1.ListTaskPushNotificationConfigRequest) (*a2av1.ListTaskPushNotificationConfigResponse, error)
	deletePushConfig func(context.Context, *a2av1.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error)
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
	if h.subscribeToTask != nil {
		return h.subscribeToTask(req, stream)
	}
	return status.Error(codes.Unimplemented, "SubscribeToTask not configured")
}

func (h *testHandler) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	if h.getExtendedCard != nil {
		return h.getExtendedCard(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "GetExtendedAgentCard not configured")
}

func (h *testHandler) SetTaskPushNotificationConfig(ctx context.Context, req *a2av1.SetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	if h.setPushConfig != nil {
		return h.setPushConfig(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "SetTaskPushNotificationConfig not configured")
}

func (h *testHandler) GetTaskPushNotificationConfig(ctx context.Context, req *a2av1.GetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	if h.getPushConfig != nil {
		return h.getPushConfig(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "GetTaskPushNotificationConfig not configured")
}

func (h *testHandler) ListTaskPushNotificationConfig(ctx context.Context, req *a2av1.ListTaskPushNotificationConfigRequest) (*a2av1.ListTaskPushNotificationConfigResponse, error) {
	if h.listPushConfig != nil {
		return h.listPushConfig(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "ListTaskPushNotificationConfig not configured")
}

func (h *testHandler) DeleteTaskPushNotificationConfig(ctx context.Context, req *a2av1.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	if h.deletePushConfig != nil {
		return h.deletePushConfig(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "DeleteTaskPushNotificationConfig not configured")
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
			if got := req.GetRequest().GetParts()[0].GetText(); got != "ping" {
				t.Fatalf("expected ping, got %q", got)
			}
			msg := &a2av1.Message{
				Role:      a2av1.Role_ROLE_AGENT,
				Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ok"}}},
				ContextId: req.GetRequest().GetContextId(),
				TaskId:    req.GetRequest().GetTaskId(),
			}
			return &a2av1.SendMessageResponse{
				Payload: &a2av1.SendMessageResponse_Msg{Msg: msg},
			}, nil
		},
	}
	srv := New(handler)
	payload, err := protojson.Marshal(&a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ping"}}},
			ContextId: "ctx-1",
			TaskId:    "task-1",
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/message:send", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp a2av1.SendMessageResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got := resp.GetMsg().GetParts()[0].GetText(); got != "ok" {
		t.Fatalf("expected ok, got %q", got)
	}
}

func TestServerSendMessageEmptyBody(t *testing.T) {
	srv := New(&testHandler{})
	req := httptest.NewRequest(http.MethodPost, "/message:send", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
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
	payload, err := protojson.Marshal(&a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			Role:  a2av1.Role_ROLE_USER,
			Parts: []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ping"}}},
		},
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/message:stream", bytes.NewReader(payload))
	rec := newStreamRecorder()
	srv.ServeHTTP(rec, req)
	if rec.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.status)
	}
	if !strings.Contains(rec.body.String(), "data:") {
		t.Fatalf("expected stream data, got %q", rec.body.String())
	}
}

func TestServerListPushConfigs(t *testing.T) {
	handler := &testHandler{
		listPushConfig: func(ctx context.Context, req *a2av1.ListTaskPushNotificationConfigRequest) (*a2av1.ListTaskPushNotificationConfigResponse, error) {
			if req.GetParent() != "tasks/task-1" {
				t.Fatalf("expected parent tasks/task-1, got %q", req.GetParent())
			}
			return &a2av1.ListTaskPushNotificationConfigResponse{
				Configs: []*a2av1.TaskPushNotificationConfig{
					{Name: "tasks/task-1/pushNotificationConfigs/cfg-1"},
				},
			}, nil
		},
	}
	var _ server.PushNotificationHandler = handler
	srv := New(handler)
	req := httptest.NewRequest(http.MethodGet, "/tasks/task-1/pushNotificationConfigs", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp a2av1.ListTaskPushNotificationConfigResponse
	if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(resp.Configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(resp.Configs))
	}
}

func TestServerPushConfigsUnsupported(t *testing.T) {
	srv := New(&testHandler{})
	req := httptest.NewRequest(http.MethodGet, "/tasks/task-1/pushNotificationConfigs", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", rec.Code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if payload["title"] != codes.Unimplemented.String() {
		t.Fatalf("expected unimplemented error, got %v", payload["title"])
	}
}
