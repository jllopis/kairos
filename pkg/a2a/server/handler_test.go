package server

import (
	"context"
	"testing"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/metadata"
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
