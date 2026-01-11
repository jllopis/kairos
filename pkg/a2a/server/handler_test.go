package server

import (
	"context"
	"fmt"
	"testing"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type streamRecorder struct {
	ctx    context.Context
	sent   []*a2av1.StreamResponse
	closed bool
}

func newStreamRecorder() *streamRecorder {
	return &streamRecorder{ctx: context.Background()}
}

func (s *streamRecorder) Send(resp *a2av1.StreamResponse) error {
	s.sent = append(s.sent, resp)
	return nil
}

func (s *streamRecorder) SetHeader(metadata.MD) error  { return nil }
func (s *streamRecorder) SendHeader(metadata.MD) error { return nil }
func (s *streamRecorder) SetTrailer(metadata.MD)       {}
func (s *streamRecorder) Context() context.Context     { return s.ctx }
func (s *streamRecorder) SendMsg(any) error            { return nil }
func (s *streamRecorder) RecvMsg(any) error            { return nil }

type stubExecutor struct {
	Output    any
	Artifacts []*a2av1.Artifact
	Err       error
}

func boolPtr(value bool) *bool {
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

	if len(stream.sent) != 3 {
		t.Fatalf("expected 3 stream responses, got %d", len(stream.sent))
	}
	if stream.sent[0].GetTask() == nil {
		t.Fatalf("expected task as first stream response")
	}
	if stream.sent[1].GetMsg() == nil {
		t.Fatalf("expected message as second stream response")
	}
	status := stream.sent[2].GetStatusUpdate()
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
	if len(stream.sent) != 1 {
		t.Fatalf("expected 1 stream response, got %d", len(stream.sent))
	}
	event := stream.sent[0].GetStatusUpdate()
	if event == nil || !event.Final {
		t.Fatalf("expected final status update")
	}
	if event.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_COMPLETED {
		t.Fatalf("expected completed state, got %v", event.GetStatus().GetState())
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
