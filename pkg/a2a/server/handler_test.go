package server

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var errStub = fmt.Errorf("executor error")

type streamRecorder struct {
	mu     sync.Mutex
	ctx    context.Context
	sent   []*a2av1.StreamResponse
	closed bool
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{ctx: context.Background()}
}

func (s *streamRecorder) Send(resp *a2av1.StreamResponse) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sent = append(s.sent, resp)
	return nil
}

func (s *streamRecorder) SetHeader(metadata.MD) error  { return nil }
func (s *streamRecorder) SendHeader(metadata.MD) error { return nil }
func (s *streamRecorder) SetTrailer(metadata.MD)       {}
func (s *streamRecorder) Context() context.Context     { return s.ctx }
func (s *streamRecorder) SendMsg(any) error            { return nil }
func (s *streamRecorder) RecvMsg(any) error            { return nil }

func (s *streamRecorder) snapshot() []*a2av1.StreamResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*a2av1.StreamResponse, len(s.sent))
	copy(out, s.sent)
	return out
}

type stubExecutor struct {
	Output    any
	Artifacts []*a2av1.Artifact
	Err       error
}

func boolPtr(value bool) *bool {
	return &value
}

func int32Ptr(value int32) *int32 {
	return &value
}

func (s *stubExecutor) Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error) {
	return s.Output, s.Artifacts, s.Err
}

func TestSendStreamingMessage_Order(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
		Card: &a2av1.AgentCard{
			Capabilities: &a2av1.AgentCapabilities{Streaming: boolPtr(true)},
		},
	}

	req := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-1",
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
		},
	}

	stream := newStreamRecorder()
	if err := handler.SendStreamingMessage(req, stream); err != nil {
		t.Fatalf("SendStreamingMessage error: %v", err)
	}

	responses := stream.snapshot()
	if len(responses) != 3 {
		t.Fatalf("expected 3 stream responses, got %d", len(responses))
	}
	if responses[0].GetTask() == nil {
		t.Fatalf("expected task as first stream response")
	}
	if responses[1].GetMsg() == nil {
		t.Fatalf("expected message as second stream response")
	}
	status := responses[2].GetStatusUpdate()
	if status == nil || !status.Final {
		t.Fatalf("expected final status update as third response")
	}
}

func TestCancelTask(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	cancelled, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("CancelTask error: %v", err)
	}
	if cancelled.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_CANCELLED {
		t.Fatalf("expected cancelled state, got %v", cancelled.GetStatus().GetState())
	}
}

func TestSubscribeToTask_TerminalStatus(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	status := newStatus(a2av1.TaskState_TASK_STATE_COMPLETED, task.History[0])
	if err := store.UpdateStatus(context.Background(), task.Id, status); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	handler := &SimpleHandler{Store: store}
	stream := newStreamRecorder()

	req := &a2av1.SubscribeToTaskRequest{Name: fmt.Sprintf("tasks/%s", task.Id)}
	if err := handler.SubscribeToTask(req, stream); err != nil {
		t.Fatalf("SubscribeToTask error: %v", err)
	}
	responses := stream.snapshot()
	if len(responses) != 1 {
		t.Fatalf("expected 1 stream response, got %d", len(responses))
	}
	event := responses[0].GetStatusUpdate()
	if event == nil || !event.Final {
		t.Fatalf("expected final status update")
	}
	if event.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_COMPLETED {
		t.Fatalf("expected completed state, got %v", event.GetStatus().GetState())
	}
}

func TestSubscribeToTask_TerminalStatusWithArtifacts(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), task.Id, []*a2av1.Artifact{
		{Name: "result", Parts: []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "done"}}}},
	}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}
	status := newStatus(a2av1.TaskState_TASK_STATE_COMPLETED, task.History[0])
	if err := store.UpdateStatus(context.Background(), task.Id, status); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	handler := &SimpleHandler{Store: store}
	stream := newStreamRecorder()

	req := &a2av1.SubscribeToTaskRequest{Name: fmt.Sprintf("tasks/%s", task.Id)}
	if err := handler.SubscribeToTask(req, stream); err != nil {
		t.Fatalf("SubscribeToTask error: %v", err)
	}
	responses := stream.snapshot()
	if len(responses) != 1 {
		t.Fatalf("expected 1 stream response, got %d", len(responses))
	}
	event := responses[0].GetStatusUpdate()
	if event == nil || !event.Final {
		t.Fatalf("expected final status update")
	}
	if event.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_COMPLETED {
		t.Fatalf("expected completed state, got %v", event.GetStatus().GetState())
	}
	if responses[0].GetArtifactUpdate() != nil {
		t.Fatalf("expected no artifact update for terminal snapshot")
	}
}

func TestPushNotificationConfigCRUD(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	handler := &SimpleHandler{
		Store:    store,
		PushCfgs: NewMemoryPushConfigStore(),
	}

	setReq := &a2av1.SetTaskPushNotificationConfigRequest{
		Parent:   fmt.Sprintf("tasks/%s", task.Id),
		ConfigId: "cfg-1",
		Config:   &a2av1.TaskPushNotificationConfig{PushNotificationConfig: &a2av1.PushNotificationConfig{Url: "https://example.com/hook"}},
	}
	cfg, err := handler.SetTaskPushNotificationConfig(context.Background(), setReq)
	if err != nil {
		t.Fatalf("SetTaskPushNotificationConfig error: %v", err)
	}
	if cfg.GetName() == "" || cfg.GetPushNotificationConfig().GetId() != "cfg-1" {
		t.Fatalf("expected config with name and id")
	}

	getReq := &a2av1.GetTaskPushNotificationConfigRequest{Name: cfg.GetName()}
	got, err := handler.GetTaskPushNotificationConfig(context.Background(), getReq)
	if err != nil {
		t.Fatalf("GetTaskPushNotificationConfig error: %v", err)
	}
	if got.GetName() != cfg.GetName() {
		t.Fatalf("expected config name %q, got %q", cfg.GetName(), got.GetName())
	}

	listReq := &a2av1.ListTaskPushNotificationConfigRequest{Parent: fmt.Sprintf("tasks/%s", task.Id), PageSize: 1}
	listed, err := handler.ListTaskPushNotificationConfig(context.Background(), listReq)
	if err != nil {
		t.Fatalf("ListTaskPushNotificationConfig error: %v", err)
	}
	if len(listed.GetConfigs()) != 1 {
		t.Fatalf("expected 1 config, got %d", len(listed.GetConfigs()))
	}

	delReq := &a2av1.DeleteTaskPushNotificationConfigRequest{Name: cfg.GetName()}
	if _, err := handler.DeleteTaskPushNotificationConfig(context.Background(), delReq); err != nil {
		t.Fatalf("DeleteTaskPushNotificationConfig error: %v", err)
	}
}

func TestPushConfig_MismatchedTaskName(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	handler := &SimpleHandler{
		Store:    store,
		PushCfgs: NewMemoryPushConfigStore(),
	}

	req := &a2av1.SetTaskPushNotificationConfigRequest{
		Parent:   fmt.Sprintf("tasks/%s", task.Id),
		ConfigId: "cfg-1",
		Config: &a2av1.TaskPushNotificationConfig{
			Name: "tasks/other/pushNotificationConfigs/cfg-1",
			PushNotificationConfig: &a2av1.PushNotificationConfig{
				Url: "https://example.com/hook",
			},
		},
	}
	if _, err := handler.SetTaskPushNotificationConfig(context.Background(), req); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestPushConfig_ConfigIDMismatch(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	handler := &SimpleHandler{
		Store:    store,
		PushCfgs: NewMemoryPushConfigStore(),
	}

	req := &a2av1.SetTaskPushNotificationConfigRequest{
		Parent:   fmt.Sprintf("tasks/%s", task.Id),
		ConfigId: "cfg-1",
		Config: &a2av1.TaskPushNotificationConfig{
			Name: "tasks/" + task.Id + "/pushNotificationConfigs/cfg-2",
			PushNotificationConfig: &a2av1.PushNotificationConfig{
				Url: "https://example.com/hook",
			},
		},
	}
	if _, err := handler.SetTaskPushNotificationConfig(context.Background(), req); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestPushConfig_ConfigIDFromPayload(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	handler := &SimpleHandler{
		Store:    store,
		PushCfgs: NewMemoryPushConfigStore(),
	}

	req := &a2av1.SetTaskPushNotificationConfigRequest{
		Parent: "tasks/" + task.Id,
		Config: &a2av1.TaskPushNotificationConfig{
			PushNotificationConfig: &a2av1.PushNotificationConfig{
				Id:  "cfg-1",
				Url: "https://example.com/hook",
			},
		},
	}
	cfg, err := handler.SetTaskPushNotificationConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetTaskPushNotificationConfig error: %v", err)
	}
	if cfg.GetPushNotificationConfig().GetId() != "cfg-1" {
		t.Fatalf("expected config id from payload")
	}
	if cfg.GetName() != "tasks/"+task.Id+"/pushNotificationConfigs/cfg-1" {
		t.Fatalf("unexpected config name %q", cfg.GetName())
	}
}

func TestPushConfig_GeneratesConfigID(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	handler := &SimpleHandler{
		Store:    store,
		PushCfgs: NewMemoryPushConfigStore(),
	}

	req := &a2av1.SetTaskPushNotificationConfigRequest{
		Parent: "tasks/" + task.Id,
		Config: &a2av1.TaskPushNotificationConfig{
			PushNotificationConfig: &a2av1.PushNotificationConfig{
				Url: "https://example.com/hook",
			},
		},
	}
	cfg, err := handler.SetTaskPushNotificationConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetTaskPushNotificationConfig error: %v", err)
	}
	if cfg.GetPushNotificationConfig().GetId() == "" {
		t.Fatalf("expected generated config id")
	}
	if cfg.GetName() == "" {
		t.Fatalf("expected generated config name")
	}
}

func TestPushConfig_ConfigIDFromRequest(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	handler := &SimpleHandler{
		Store:    store,
		PushCfgs: NewMemoryPushConfigStore(),
	}

	req := &a2av1.SetTaskPushNotificationConfigRequest{
		Parent:   "tasks/" + task.Id,
		ConfigId: "cfg-1",
		Config: &a2av1.TaskPushNotificationConfig{
			PushNotificationConfig: &a2av1.PushNotificationConfig{
				Url: "https://example.com/hook",
			},
		},
	}
	cfg, err := handler.SetTaskPushNotificationConfig(context.Background(), req)
	if err != nil {
		t.Fatalf("SetTaskPushNotificationConfig error: %v", err)
	}
	if cfg.GetPushNotificationConfig().GetId() != "cfg-1" {
		t.Fatalf("expected config id from request")
	}
	if cfg.GetName() != "tasks/"+task.Id+"/pushNotificationConfigs/cfg-1" {
		t.Fatalf("unexpected config name %q", cfg.GetName())
	}
}

func TestSubscribeToTask_UpdatesAndArtifacts(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	handler := &SimpleHandler{Store: store}
	stream := newStreamRecorder()

	ctx, cancel := context.WithCancel(context.Background())
	stream.ctx = ctx

	done := make(chan error, 1)
	go func() {
		req := &a2av1.SubscribeToTaskRequest{Name: fmt.Sprintf("tasks/%s", task.Id)}
		done <- handler.SubscribeToTask(req, stream)
	}()

	waitFor := func(cond func() bool) bool {
		deadline := time.Now().Add(1 * time.Second)
		for time.Now().Before(deadline) {
			if cond() {
				return true
			}
			time.Sleep(20 * time.Millisecond)
		}
		return false
	}

	if ok := waitFor(func() bool { return len(stream.snapshot()) > 0 }); !ok {
		t.Fatalf("expected initial status update")
	}

	working := newStatus(a2av1.TaskState_TASK_STATE_WORKING, task.History[0])
	if err := store.UpdateStatus(context.Background(), task.Id, working); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if ok := waitFor(func() bool {
		for _, resp := range stream.snapshot() {
			if resp.GetStatusUpdate() != nil && resp.GetStatusUpdate().GetStatus().GetState() == a2av1.TaskState_TASK_STATE_WORKING {
				return true
			}
		}
		return false
	}); !ok {
		t.Fatalf("expected working status update")
	}

	artifact := &a2av1.Artifact{Name: "result", Parts: []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "done"}}}}
	if err := store.AddArtifacts(context.Background(), task.Id, []*a2av1.Artifact{artifact}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	if ok := waitFor(func() bool {
		for _, resp := range stream.snapshot() {
			if resp.GetArtifactUpdate() != nil {
				return true
			}
		}
		return false
	}); !ok {
		t.Fatalf("expected artifact update")
	}

	cancel()
	<-done
}

func TestGetExtendedAgentCard_NotSupported(t *testing.T) {
	handler := &SimpleHandler{}

	if _, err := handler.GetExtendedAgentCard(context.Background(), &a2av1.GetExtendedAgentCardRequest{}); status.Code(err) != codes.Unimplemented {
		t.Fatalf("expected Unimplemented, got %v", status.Code(err))
	}
}

func TestGetExtendedAgentCard_MissingSkills(t *testing.T) {
	handler := &SimpleHandler{
		Card: &a2av1.AgentCard{
			SupportsExtendedAgentCard: boolPtr(true),
		},
	}

	if _, err := handler.GetExtendedAgentCard(context.Background(), &a2av1.GetExtendedAgentCardRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func TestGetExtendedAgentCard_OK(t *testing.T) {
	handler := &SimpleHandler{
		Card: &a2av1.AgentCard{
			SupportsExtendedAgentCard: boolPtr(true),
			Skills:                    []*a2av1.AgentSkill{{Id: "echo", Name: "Echo"}},
		},
	}

	card, err := handler.GetExtendedAgentCard(context.Background(), &a2av1.GetExtendedAgentCardRequest{})
	if err != nil {
		t.Fatalf("GetExtendedAgentCard error: %v", err)
	}
	if len(card.GetSkills()) == 0 {
		t.Fatalf("expected skills in card")
	}
}

func TestSetTaskPushNotificationConfig_InvalidParent(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		PushCfgs: NewMemoryPushConfigStore(),
	}
	req := &a2av1.SetTaskPushNotificationConfigRequest{
		Parent: "tasks/",
		Config: &a2av1.TaskPushNotificationConfig{
			PushNotificationConfig: &a2av1.PushNotificationConfig{Url: "https://example.com/hook"},
		},
	}
	if _, err := handler.SetTaskPushNotificationConfig(context.Background(), req); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestSetTaskPushNotificationConfig_MissingConfig(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	handler := &SimpleHandler{
		Store:    store,
		PushCfgs: NewMemoryPushConfigStore(),
	}
	req := &a2av1.SetTaskPushNotificationConfigRequest{Parent: fmt.Sprintf("tasks/%s", task.Id)}
	if _, err := handler.SetTaskPushNotificationConfig(context.Background(), req); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestSubscribeToTask_InvalidName(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}
	req := &a2av1.SubscribeToTaskRequest{Name: "tasks/"}
	stream := newStreamRecorder()

	if err := handler.SubscribeToTask(req, stream); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestSubscribeToTask_PlainIDName(t *testing.T) {
	store := NewMemoryTaskStore()
	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), task.Id, newStatus(a2av1.TaskState_TASK_STATE_COMPLETED, task.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	handler := &SimpleHandler{Store: store}
	stream := newStreamRecorder()

	req := &a2av1.SubscribeToTaskRequest{Name: task.Id}
	if err := handler.SubscribeToTask(req, stream); err != nil {
		t.Fatalf("SubscribeToTask error: %v", err)
	}
	if len(stream.snapshot()) != 1 {
		t.Fatalf("expected 1 stream response, got %d", len(stream.snapshot()))
	}
}

func TestPushConfig_InvalidNames(t *testing.T) {
	handler := &SimpleHandler{PushCfgs: NewMemoryPushConfigStore()}

	_, err := handler.GetTaskPushNotificationConfig(context.Background(), &a2av1.GetTaskPushNotificationConfigRequest{Name: "bad"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for Get, got %v", status.Code(err))
	}

	_, err = handler.ListTaskPushNotificationConfig(context.Background(), &a2av1.ListTaskPushNotificationConfigRequest{Parent: "tasks/"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for List, got %v", status.Code(err))
	}

	_, err = handler.DeleteTaskPushNotificationConfig(context.Background(), &a2av1.DeleteTaskPushNotificationConfigRequest{Name: "tasks/abc/invalid"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for Delete, got %v", status.Code(err))
	}
}

func TestPushConfig_PlainNameRejected(t *testing.T) {
	handler := &SimpleHandler{PushCfgs: NewMemoryPushConfigStore()}

	_, err := handler.GetTaskPushNotificationConfig(context.Background(), &a2av1.GetTaskPushNotificationConfigRequest{Name: "config-1"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for plain name, got %v", status.Code(err))
	}

	_, err = handler.DeleteTaskPushNotificationConfig(context.Background(), &a2av1.DeleteTaskPushNotificationConfigRequest{Name: "config-1"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for plain name, got %v", status.Code(err))
	}
}

func TestTaskOps_InvalidNames(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{Name: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for GetTask, got %v", status.Code(err))
	}

	_, err = handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: ""})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for CancelTask, got %v", status.Code(err))
	}
}

func TestSendMessage_InvalidMessage(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	_, err := handler.SendMessage(context.Background(), &a2av1.SendMessageRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for missing message, got %v", status.Code(err))
	}

	_, err = handler.SendMessage(context.Background(), &a2av1.SendMessageRequest{
		Request: &a2av1.Message{MessageId: "msg-1"},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for missing role/parts, got %v", status.Code(err))
	}
}

func TestSendStreamingMessage_InvalidMessage(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
		Card: &a2av1.AgentCard{
			Capabilities: &a2av1.AgentCapabilities{Streaming: boolPtr(true)},
		},
	}
	stream := newStreamRecorder()

	if err := handler.SendStreamingMessage(&a2av1.SendMessageRequest{}, stream); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for missing message, got %v", status.Code(err))
	}

	if err := handler.SendStreamingMessage(&a2av1.SendMessageRequest{
		Request: &a2av1.Message{MessageId: "msg-1"},
	}, stream); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument for missing role/parts, got %v", status.Code(err))
	}
}

func TestCancelTask_Idempotent(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	cancelled := newStatus(a2av1.TaskState_TASK_STATE_CANCELLED, task.History[0])
	if err := handler.Store.UpdateStatus(context.Background(), task.Id, cancelled); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	out, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("CancelTask error: %v", err)
	}
	if out.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_CANCELLED {
		t.Fatalf("expected cancelled state, got %v", out.GetStatus().GetState())
	}
}

func TestCancelTask_DoesNotOverrideCompleted(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	completed := newStatus(a2av1.TaskState_TASK_STATE_COMPLETED, task.History[0])
	if err := handler.Store.UpdateStatus(context.Background(), task.Id, completed); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	out, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("CancelTask error: %v", err)
	}
	if out.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_COMPLETED {
		t.Fatalf("expected completed state, got %v", out.GetStatus().GetState())
	}
}

func TestCancelTask_DoesNotOverrideFailed(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	failed := newStatus(a2av1.TaskState_TASK_STATE_FAILED, task.History[0])
	if err := handler.Store.UpdateStatus(context.Background(), task.Id, failed); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	out, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("CancelTask error: %v", err)
	}
	if out.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_FAILED {
		t.Fatalf("expected failed state, got %v", out.GetStatus().GetState())
	}
}

func TestCancelTask_DoesNotOverrideRejected(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	rejected := newStatus(a2av1.TaskState_TASK_STATE_REJECTED, task.History[0])
	if err := handler.Store.UpdateStatus(context.Background(), task.Id, rejected); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	out, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("CancelTask error: %v", err)
	}
	if out.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_REJECTED {
		t.Fatalf("expected rejected state, got %v", out.GetStatus().GetState())
	}
}

func TestSendMessage_BlockingAndAsync(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	blockingReq := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-1",
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
		},
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	}
	resp, err := handler.SendMessage(context.Background(), blockingReq)
	if err != nil {
		t.Fatalf("SendMessage blocking error: %v", err)
	}
	if resp.GetMsg() == nil || resp.GetTask() != nil {
		t.Fatalf("expected message response for blocking call")
	}

	asyncReq := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-2",
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hi"}}},
		},
		Configuration: &a2av1.SendMessageConfiguration{Blocking: false},
	}
	resp, err = handler.SendMessage(context.Background(), asyncReq)
	if err != nil {
		t.Fatalf("SendMessage async error: %v", err)
	}
	if resp.GetTask() == nil || resp.GetMsg() != nil {
		t.Fatalf("expected task response for async call")
	}
}

func TestSendMessage_AppendsHistoryForExistingTask(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	req := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-2",
			TaskId:    task.Id,
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
		},
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	}
	if _, err := handler.SendMessage(context.Background(), req); err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}

	updated, err := handler.Store.GetTask(context.Background(), task.Id, 0, true)
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got := len(updated.GetHistory()); got < 2 {
		t.Fatalf("expected history to grow, got %d", got)
	}
	if updated.GetHistory()[len(updated.GetHistory())-2].GetMessageId() != "msg-2" {
		t.Fatalf("expected appended message in history")
	}
}

func TestSendMessage_RejectsTerminalTask(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	completed := newStatus(a2av1.TaskState_TASK_STATE_COMPLETED, task.History[0])
	if err := handler.Store.UpdateStatus(context.Background(), task.Id, completed); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	req := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-2",
			TaskId:    task.Id,
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
		},
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	}
	if _, err := handler.SendMessage(context.Background(), req); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", status.Code(err))
	}
}

func TestSendMessage_PreservesContextID(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	task, err := handler.Store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	req := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-2",
			TaskId:    task.Id,
			ContextId: "custom",
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
		},
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	}
	if _, err := handler.SendMessage(context.Background(), req); err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}

	updated, err := handler.Store.GetTask(context.Background(), task.Id, 0, true)
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	last := updated.GetHistory()[len(updated.GetHistory())-2]
	if last.GetContextId() != updated.GetContextId() {
		t.Fatalf("expected message context id to match task context id")
	}
	if last.GetContextId() == "custom" {
		t.Fatalf("expected task context to override custom context id")
	}
}

func TestSendMessage_TaskNotFound(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Output: "ok"},
	}

	req := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-1",
			TaskId:    "missing",
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
		},
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	}
	if _, err := handler.SendMessage(context.Background(), req); status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestSendMessage_ExecutorError(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Err: errStub},
	}

	req := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-1",
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
		},
		Configuration: &a2av1.SendMessageConfiguration{Blocking: true},
	}
	if _, err := handler.SendMessage(context.Background(), req); status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}

	taskList, _, err := handler.Store.ListTasks(context.Background(), TaskFilter{})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(taskList) == 0 {
		t.Fatalf("expected task to be stored")
	}
	if taskList[0].GetStatus().GetState() != a2av1.TaskState_TASK_STATE_FAILED {
		t.Fatalf("expected failed task status, got %v", taskList[0].GetStatus().GetState())
	}
}

func TestSendStreamingMessage_ExecutorError(t *testing.T) {
	handler := &SimpleHandler{
		Store:    NewMemoryTaskStore(),
		Executor: &stubExecutor{Err: errStub},
		Card: &a2av1.AgentCard{
			Capabilities: &a2av1.AgentCapabilities{Streaming: boolPtr(true)},
		},
	}
	stream := newStreamRecorder()

	req := &a2av1.SendMessageRequest{
		Request: &a2av1.Message{
			MessageId: "msg-1",
			Role:      a2av1.Role_ROLE_USER,
			Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
		},
	}
	if err := handler.SendStreamingMessage(req, stream); status.Code(err) != codes.Internal {
		t.Fatalf("expected Internal, got %v", status.Code(err))
	}

	taskList, _, err := handler.Store.ListTasks(context.Background(), TaskFilter{})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(taskList) == 0 {
		t.Fatalf("expected task to be stored")
	}
	if taskList[0].GetStatus().GetState() != a2av1.TaskState_TASK_STATE_FAILED {
		t.Fatalf("expected failed task status, got %v", taskList[0].GetStatus().GetState())
	}
}

func TestListTasks_Filtering(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-b",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskB.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskB.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskA.Id, []*a2av1.Artifact{{Name: "artifact"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{ContextId: "ctx-a", IncludeArtifacts: boolPtr(true)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task for context, got %d", len(resp.GetTasks()))
	}
	if resp.GetTasks()[0].GetContextId() != "ctx-a" {
		t.Fatalf("expected ctx-a")
	}
	if len(resp.GetTasks()[0].GetArtifacts()) == 0 {
		t.Fatalf("expected artifacts included")
	}

	resp, err = handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{Status: a2av1.TaskState_TASK_STATE_WORKING})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 || resp.GetTasks()[0].GetId() != taskB.Id {
		t.Fatalf("expected working task")
	}

	resp, err = handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{ContextId: "ctx-a", IncludeArtifacts: boolPtr(false)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task")
	}
	if len(resp.GetTasks()[0].GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped when include_artifacts is false")
	}
}

func TestListTasks_HistoryLength(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{HistoryLength: int32Ptr(1)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.GetTasks()))
	}
	if got := len(resp.GetTasks()[0].GetHistory()); got != 1 {
		t.Fatalf("expected history length 1, got %d", got)
	}
	if resp.GetTasks()[0].GetHistory()[0].GetMessageId() != "msg-2" {
		t.Fatalf("expected latest history item")
	}
}

func TestListTasks_PageSize(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(1)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.GetTasks()))
	}
	if resp.GetPageSize() != 1 {
		t.Fatalf("expected page size 1, got %d", resp.GetPageSize())
	}
}

func TestListTasks_LastUpdatedAfter(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	later := time.Now().UTC()
	time.Sleep(10 * time.Millisecond)
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{LastUpdatedAfter: later.UnixMilli()})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.GetTasks()))
	}
	if resp.GetTasks()[0].GetHistory()[0].GetMessageId() != "msg-2" {
		t.Fatalf("expected the newer task")
	}
}

func TestGetTask_HistoryLengthAndArtifacts(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), task.Id, []*a2av1.Artifact{{Name: "artifact"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	resp, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{
		Name:          task.Id,
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got := len(resp.GetHistory()); got != 1 {
		t.Fatalf("expected history length 1, got %d", got)
	}
	if resp.GetHistory()[0].GetMessageId() != "msg-2" {
		t.Fatalf("expected latest history item")
	}
	if len(resp.GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped by default")
	}

}

func TestListTasks_PageTokenPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	first, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(1)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(first.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(first.GetTasks()))
	}
	if first.GetNextPageToken() == "" {
		t.Fatalf("expected next page token")
	}

	second, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(1), PageToken: first.GetNextPageToken()})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(second.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on second page, got %d", len(second.GetTasks()))
	}
	if first.GetTasks()[0].GetId() == second.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}

	_, err = handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageToken: "bad"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestListTasks_PageTokenBeyondRange(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageToken: "10", PageSize: int32Ptr(1)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 0 {
		t.Fatalf("expected empty page, got %d", len(resp.GetTasks()))
	}
	if resp.GetNextPageToken() != "" {
		t.Fatalf("expected no next page token")
	}
}

func TestListTasks_PageTokenNegative(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageToken: "-1"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestListTasks_PageTokenLeadingZeros(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	first, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(1)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(first.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(first.GetTasks()))
	}

	second, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(1), PageToken: "01"})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(second.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(second.GetTasks()))
	}
	if first.GetTasks()[0].GetId() == second.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_PageTokenWhitespace(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageToken: " 1"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestListTasks_PageTokenLargeOffset(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageToken: "999999", PageSize: int32Ptr(1)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 0 {
		t.Fatalf("expected empty page, got %d", len(resp.GetTasks()))
	}
	if resp.GetNextPageToken() != "" {
		t.Fatalf("expected no next page token")
	}
}

func TestListTasks_DefaultPageSize(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(0)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if resp.GetPageSize() != 50 {
		t.Fatalf("expected default page size 50, got %d", resp.GetPageSize())
	}
}

func TestListTasks_DefaultPageSizeWithPageToken(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(0), PageToken: "1"})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if resp.GetPageSize() != 50 {
		t.Fatalf("expected default page size 50, got %d", resp.GetPageSize())
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page, got %d", len(resp.GetTasks()))
	}
}

func TestListTasks_DefaultPageSizeWithPageTokenBeyondRange(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(0), PageToken: "5"})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if resp.GetPageSize() != 50 {
		t.Fatalf("expected default page size 50, got %d", resp.GetPageSize())
	}
	if len(resp.GetTasks()) != 0 {
		t.Fatalf("expected empty page, got %d", len(resp.GetTasks()))
	}
	if resp.GetNextPageToken() != "" {
		t.Fatalf("expected no next page token")
	}
}

func TestListTasks_OrderingAcrossPages(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	firstTask, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	secondTask, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(1)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(page1.GetTasks()))
	}
	if page1.GetTasks()[0].GetId() != secondTask.Id {
		t.Fatalf("expected newer task first")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(1), PageToken: page1.GetNextPageToken()})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if page2.GetTasks()[0].GetId() != firstTask.Id {
		t.Fatalf("expected older task on page 2")
	}
}

func TestListTasks_OrderingWithEqualTimestamps(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	firstTask, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	secondTask, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	fixed := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	store.mu.Lock()
	store.tasks[firstTask.Id].updatedAt = fixed
	store.tasks[secondTask.Id].updatedAt = fixed
	store.mu.Unlock()

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(2)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.GetTasks()))
	}

	expectedFirst := firstTask.Id
	expectedSecond := secondTask.Id
	if expectedSecond < expectedFirst {
		expectedFirst, expectedSecond = expectedSecond, expectedFirst
	}
	if resp.GetTasks()[0].GetId() != expectedFirst || resp.GetTasks()[1].GetId() != expectedSecond {
		t.Fatalf("expected id order %s then %s", expectedFirst, expectedSecond)
	}
}

func TestListTasks_OrderingByUpdatedAt(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := store.UpdateStatus(context.Background(), taskA.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskA.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(2)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.GetTasks()))
	}
	if resp.GetTasks()[0].GetId() != taskA.Id {
		t.Fatalf("expected updated task first")
	}
	if resp.GetTasks()[1].GetId() != taskB.Id {
		t.Fatalf("expected other task second")
	}
}

func TestListTasks_OrderingByArtifactUpdate(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := store.AddArtifacts(context.Background(), taskB.Id, []*a2av1.Artifact{{Name: "artifact"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(2)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.GetTasks()))
	}
	if resp.GetTasks()[0].GetId() != taskB.Id {
		t.Fatalf("expected artifact-updated task first")
	}
	if resp.GetTasks()[1].GetId() != taskA.Id {
		t.Fatalf("expected other task second")
	}
}

func TestListTasks_OrderingByHistoryAppend(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	time.Sleep(5 * time.Millisecond)
	if err := store.AppendHistory(context.Background(), taskA.Id, &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(2)})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(resp.GetTasks()))
	}
	if resp.GetTasks()[0].GetId() != taskA.Id {
		t.Fatalf("expected history-updated task first")
	}
	if resp.GetTasks()[1].GetId() != taskB.Id {
		t.Fatalf("expected other task second")
	}
}

func TestListTasks_StatusFilterWithPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskA.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskA.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskB.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskB.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		Status:   a2av1.TaskState_TASK_STATE_WORKING,
		PageSize: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if page1.GetNextPageToken() == "" {
		t.Fatalf("expected next page token")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		Status:    a2av1.TaskState_TASK_STATE_WORKING,
		PageSize:  int32Ptr(1),
		PageToken: page1.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_ContextFilterWithPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	_, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-b",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "gamma"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId: "ctx-a",
		PageSize:  int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if page1.GetTasks()[0].GetContextId() != "ctx-a" {
		t.Fatalf("expected ctx-a task")
	}
	if page1.GetNextPageToken() == "" {
		t.Fatalf("expected next page token")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId: "ctx-a",
		PageSize:  int32Ptr(1),
		PageToken: page1.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if page2.GetTasks()[0].GetContextId() != "ctx-a" {
		t.Fatalf("expected ctx-a task")
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_ContextAndStatusFiltersWithPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskA.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskA.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskB.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskB.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	_, err = store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-b",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "gamma"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId: "ctx-a",
		Status:    a2av1.TaskState_TASK_STATE_WORKING,
		PageSize:  int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if page1.GetNextPageToken() == "" {
		t.Fatalf("expected next page token")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId: "ctx-a",
		Status:    a2av1.TaskState_TASK_STATE_WORKING,
		PageSize:  int32Ptr(1),
		PageToken: page1.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_IncludeArtifactsWithPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskA.Id, []*a2av1.Artifact{{Name: "artifact-a"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskB.Id, []*a2av1.Artifact{{Name: "artifact-b"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		PageSize:         int32Ptr(1),
		IncludeArtifacts: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if len(page1.GetTasks()[0].GetArtifacts()) == 0 {
		t.Fatalf("expected artifacts on page 1")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		PageSize:         int32Ptr(1),
		PageToken:        page1.GetNextPageToken(),
		IncludeArtifacts: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if len(page2.GetTasks()[0].GetArtifacts()) == 0 {
		t.Fatalf("expected artifacts on page 2")
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_ExcludeArtifactsWithPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskA.Id, []*a2av1.Artifact{{Name: "artifact-a"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskB.Id, []*a2av1.Artifact{{Name: "artifact-b"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		PageSize:         int32Ptr(1),
		IncludeArtifacts: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if len(page1.GetTasks()[0].GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped on page 1")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		PageSize:         int32Ptr(1),
		PageToken:        page1.GetNextPageToken(),
		IncludeArtifacts: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if len(page2.GetTasks()[0].GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped on page 2")
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_FiltersWithArtifactsPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskC, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-b",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "gamma"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskA.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskA.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskB.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskB.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskC.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskC.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskA.Id, []*a2av1.Artifact{{Name: "artifact-a"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskB.Id, []*a2av1.Artifact{{Name: "artifact-b"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		Status:           a2av1.TaskState_TASK_STATE_WORKING,
		PageSize:         int32Ptr(1),
		IncludeArtifacts: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if len(page1.GetTasks()[0].GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped on page 1")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		Status:           a2av1.TaskState_TASK_STATE_WORKING,
		PageSize:         int32Ptr(1),
		PageToken:        page1.GetNextPageToken(),
		IncludeArtifacts: boolPtr(false),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if len(page2.GetTasks()[0].GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped on page 2")
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_FiltersWithArtifactsIncluded(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		ContextId: "ctx-a",
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskA.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskA.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.UpdateStatus(context.Background(), taskB.Id, newStatus(a2av1.TaskState_TASK_STATE_WORKING, taskB.History[0])); err != nil {
		t.Fatalf("UpdateStatus error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskA.Id, []*a2av1.Artifact{{Name: "artifact-a"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), taskB.Id, []*a2av1.Artifact{{Name: "artifact-b"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		Status:           a2av1.TaskState_TASK_STATE_WORKING,
		PageSize:         int32Ptr(1),
		IncludeArtifacts: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if len(page1.GetTasks()[0].GetArtifacts()) == 0 {
		t.Fatalf("expected artifacts on page 1")
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		ContextId:        "ctx-a",
		Status:           a2av1.TaskState_TASK_STATE_WORKING,
		PageSize:         int32Ptr(1),
		PageToken:        page1.GetNextPageToken(),
		IncludeArtifacts: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if len(page2.GetTasks()[0].GetArtifacts()) == 0 {
		t.Fatalf("expected artifacts on page 2")
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_HistoryLengthWithPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), taskA.Id, &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), taskB.Id, &a2av1.Message{
		MessageId: "msg-4",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if got := len(page1.GetTasks()[0].GetHistory()); got != 1 {
		t.Fatalf("expected history length 1 on page 1, got %d", got)
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		PageToken:     page1.GetNextPageToken(),
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if got := len(page2.GetTasks()[0].GetHistory()); got != 1 {
		t.Fatalf("expected history length 1 on page 2, got %d", got)
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_HistoryLengthZeroWithPagination(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), taskA.Id, &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), taskB.Id, &a2av1.Message{
		MessageId: "msg-4",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		HistoryLength: int32Ptr(0),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if got := len(page1.GetTasks()[0].GetHistory()); got < 2 {
		t.Fatalf("expected full history on page 1, got %d", got)
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		PageToken:     page1.GetNextPageToken(),
		HistoryLength: int32Ptr(0),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if got := len(page2.GetTasks()[0].GetHistory()); got < 2 {
		t.Fatalf("expected full history on page 2, got %d", got)
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_HistoryLengthExceedsHistory(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		HistoryLength: int32Ptr(10),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.GetTasks()))
	}
	if got := len(resp.GetTasks()[0].GetHistory()); got != 2 {
		t.Fatalf("expected full history length 2, got %d", got)
	}
}

func TestGetTask_HistoryLengthExceedsHistory(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	resp, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{
		Name:          task.Id,
		HistoryLength: int32Ptr(10),
	})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got := len(resp.GetHistory()); got != 2 {
		t.Fatalf("expected full history length 2, got %d", got)
	}
}

func TestGetTask_HistoryLengthZero(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	resp, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{
		Name:          task.Id,
		HistoryLength: int32Ptr(0),
	})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got := len(resp.GetHistory()); got != 2 {
		t.Fatalf("expected full history length 2, got %d", got)
	}
}

func TestGetTask_HistoryLengthOne(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	resp, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{
		Name:          task.Id,
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got := len(resp.GetHistory()); got != 1 {
		t.Fatalf("expected history length 1, got %d", got)
	}
	if resp.GetHistory()[0].GetMessageId() != "msg-2" {
		t.Fatalf("expected latest message")
	}
}

func TestGetTask_StripsArtifacts(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), task.Id, []*a2av1.Artifact{{Name: "artifact"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	resp, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if len(resp.GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped")
	}
}

func TestGetTask_HistoryLengthWithArtifacts(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), task.Id, []*a2av1.Artifact{{Name: "artifact"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	resp, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{
		Name:          task.Id,
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if got := len(resp.GetHistory()); got != 1 {
		t.Fatalf("expected history length 1, got %d", got)
	}
	if resp.GetHistory()[0].GetMessageId() != "msg-2" {
		t.Fatalf("expected latest history item")
	}
	if len(resp.GetArtifacts()) != 0 {
		t.Fatalf("expected artifacts stripped")
	}
}

func TestGetTask_PlainIDName(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("GetTask error: %v", err)
	}
	if resp.GetId() != task.Id {
		t.Fatalf("expected task id %s, got %s", task.Id, resp.GetId())
	}
}

func TestCancelTask_PlainIDName(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}

	resp, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: task.Id})
	if err != nil {
		t.Fatalf("CancelTask error: %v", err)
	}
	if resp.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_CANCELLED {
		t.Fatalf("expected cancelled state, got %v", resp.GetStatus().GetState())
	}
}
func TestGetTask_NotFound(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{Name: "tasks/missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestCancelTask_NotFound(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: "tasks/missing"})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("expected NotFound, got %v", status.Code(err))
	}
}

func TestGetTask_NegativeHistoryLength(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{
		Name:          "tasks/missing",
		HistoryLength: int32Ptr(-1),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestListTasks_NegativeHistoryLength(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{HistoryLength: int32Ptr(-1)})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestListTasks_PageSizeAndHistoryLength(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if _, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	}); err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), taskA.Id, &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.GetTasks()))
	}
	if got := len(resp.GetTasks()[0].GetHistory()); got != 1 {
		t.Fatalf("expected history length 1, got %d", got)
	}
}

func TestListTasks_HistoryLengthWithPaginationTokens(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	taskA, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	taskB, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "beta"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), taskA.Id, &a2av1.Message{
		MessageId: "msg-3",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), taskB.Id, &a2av1.Message{
		MessageId: "msg-4",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}

	page1, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page1.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 1, got %d", len(page1.GetTasks()))
	}
	if got := len(page1.GetTasks()[0].GetHistory()); got != 1 {
		t.Fatalf("expected history length 1 on page 1, got %d", got)
	}

	page2, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:      int32Ptr(1),
		PageToken:     page1.GetNextPageToken(),
		HistoryLength: int32Ptr(1),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(page2.GetTasks()) != 1 {
		t.Fatalf("expected 1 task on page 2, got %d", len(page2.GetTasks()))
	}
	if got := len(page2.GetTasks()[0].GetHistory()); got != 1 {
		t.Fatalf("expected history length 1 on page 2, got %d", got)
	}
	if page1.GetTasks()[0].GetId() == page2.GetTasks()[0].GetId() {
		t.Fatalf("expected different tasks across pages")
	}
}

func TestListTasks_ArtifactsAndHistoryLength(t *testing.T) {
	store := NewMemoryTaskStore()
	handler := &SimpleHandler{Store: store}

	task, err := store.CreateTask(context.Background(), &a2av1.Message{
		MessageId: "msg-1",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "alpha"}}},
	})
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if err := store.AppendHistory(context.Background(), task.Id, &a2av1.Message{
		MessageId: "msg-2",
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "follow-up"}}},
	}); err != nil {
		t.Fatalf("AppendHistory error: %v", err)
	}
	if err := store.AddArtifacts(context.Background(), task.Id, []*a2av1.Artifact{{Name: "artifact"}}); err != nil {
		t.Fatalf("AddArtifacts error: %v", err)
	}

	resp, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{
		PageSize:         int32Ptr(1),
		HistoryLength:    int32Ptr(1),
		IncludeArtifacts: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("ListTasks error: %v", err)
	}
	if len(resp.GetTasks()) != 1 {
		t.Fatalf("expected 1 task, got %d", len(resp.GetTasks()))
	}
	if got := len(resp.GetTasks()[0].GetHistory()); got != 1 {
		t.Fatalf("expected history length 1, got %d", got)
	}
	if len(resp.GetTasks()[0].GetArtifacts()) == 0 {
		t.Fatalf("expected artifacts included")
	}
}

func TestListTasks_NegativePageSizeRejected(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.ListTasks(context.Background(), &a2av1.ListTasksRequest{PageSize: int32Ptr(-5)})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestGetTask_InvalidName(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.GetTask(context.Background(), &a2av1.GetTaskRequest{Name: "tasks/"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}

func TestCancelTask_InvalidName(t *testing.T) {
	handler := &SimpleHandler{Store: NewMemoryTaskStore()}

	_, err := handler.CancelTask(context.Background(), &a2av1.CancelTaskRequest{Name: "tasks/"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument, got %v", status.Code(err))
	}
}
