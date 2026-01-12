package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/governance"
)

func TestMemoryApprovalStore_CRUD(t *testing.T) {
	store := NewMemoryApprovalStore()
	message := &a2av1.Message{
		MessageId: uuid.NewString(),
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "hello"}}},
	}
	record, err := store.Create(context.Background(), ApprovalRecord{
		TaskID:    "task-1",
		ContextID: "ctx-1",
		ExpiresAt: time.Now().UTC().Add(-time.Minute),
		Message:   message,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if record.ID == "" {
		t.Fatalf("expected id")
	}
	if record.Status != ApprovalStatusPending {
		t.Fatalf("expected pending status")
	}
	found, err := store.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if found.TaskID != "task-1" {
		t.Fatalf("unexpected task id: %s", found.TaskID)
	}
	if _, err := store.UpdateStatus(context.Background(), record.ID, ApprovalStatusApproved, "ok"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	updated, err := store.Get(context.Background(), record.ID)
	if err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.Status != ApprovalStatusApproved {
		t.Fatalf("expected approved")
	}
	list, err := store.List(context.Background(), ApprovalFilter{Status: ApprovalStatusApproved})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected list size 1, got %d", len(list))
	}
	expiring, err := store.List(context.Background(), ApprovalFilter{
		Status:         ApprovalStatusApproved,
		ExpiringBefore: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("list expiring: %v", err)
	}
	if len(expiring) != 1 {
		t.Fatalf("expected expiring approval")
	}
}

type approvalTestExecutor struct {
	calls int
}

func (e *approvalTestExecutor) Run(_ context.Context, _ *a2av1.Message) (any, []*a2av1.Artifact, error) {
	e.calls++
	return "ok", nil, nil
}

func TestHandlerApprovalFlow(t *testing.T) {
	ctx := context.Background()
	exec := &approvalTestExecutor{}
	handler := &SimpleHandler{
		Store:         NewMemoryTaskStore(),
		Executor:      exec,
		PolicyEngine:  governance.NewRuleSet([]governance.Rule{{Effect: "pending", Type: governance.ActionAgent}}),
		ApprovalStore: NewMemoryApprovalStore(),
	}
	msg := &a2av1.Message{
		MessageId: uuid.NewString(),
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ping"}}},
	}
	resp, err := handler.SendMessage(ctx, &a2av1.SendMessageRequest{Request: msg})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	task := resp.GetTask()
	if task == nil {
		t.Fatalf("expected task response")
	}
	if task.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_INPUT_REQUIRED {
		t.Fatalf("expected input required, got %s", task.GetStatus().GetState())
	}
	approvalID := ""
	if metadata := task.GetStatus().GetMessage().GetMetadata(); metadata != nil {
		if value, ok := metadata.Fields["approval_id"]; ok {
			approvalID = value.GetStringValue()
		}
	}
	if approvalID == "" {
		t.Fatalf("expected approval_id metadata")
	}
	if _, err := handler.Approve(ctx, approvalID, "approved"); err != nil {
		t.Fatalf("approve: %v", err)
	}
	finalTask, err := handler.Store.GetTask(ctx, task.Id, 0, true)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if finalTask.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_COMPLETED {
		t.Fatalf("expected completed, got %s", finalTask.GetStatus().GetState())
	}
	if exec.calls != 1 {
		t.Fatalf("expected executor call, got %d", exec.calls)
	}
}

func TestHandlerRejectFlow(t *testing.T) {
	ctx := context.Background()
	handler := &SimpleHandler{
		Store:         NewMemoryTaskStore(),
		Executor:      &approvalTestExecutor{},
		PolicyEngine:  governance.NewRuleSet([]governance.Rule{{Effect: "pending", Type: governance.ActionAgent}}),
		ApprovalStore: NewMemoryApprovalStore(),
	}
	msg := &a2av1.Message{
		MessageId: uuid.NewString(),
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ping"}}},
	}
	resp, err := handler.SendMessage(ctx, &a2av1.SendMessageRequest{Request: msg})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	task := resp.GetTask()
	if task == nil {
		t.Fatalf("expected task response")
	}
	approvalID := ""
	if metadata := task.GetStatus().GetMessage().GetMetadata(); metadata != nil {
		if value, ok := metadata.Fields["approval_id"]; ok {
			approvalID = value.GetStringValue()
		}
	}
	if approvalID == "" {
		t.Fatalf("expected approval_id metadata")
	}
	if _, err := handler.Reject(ctx, approvalID, "denied"); err != nil {
		t.Fatalf("reject: %v", err)
	}
	finalTask, err := handler.Store.GetTask(ctx, task.Id, 0, true)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if finalTask.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_REJECTED {
		t.Fatalf("expected rejected, got %s", finalTask.GetStatus().GetState())
	}
}

func TestHandlerExpireApprovals(t *testing.T) {
	ctx := context.Background()
	handler := &SimpleHandler{
		Store:           NewMemoryTaskStore(),
		Executor:        &approvalTestExecutor{},
		PolicyEngine:    governance.NewRuleSet([]governance.Rule{{Effect: "pending", Type: governance.ActionAgent}}),
		ApprovalStore:   NewMemoryApprovalStore(),
		ApprovalTimeout: -time.Second,
	}
	msg := &a2av1.Message{
		MessageId: uuid.NewString(),
		Role:      a2av1.Role_ROLE_USER,
		Parts:     []*a2av1.Part{{Part: &a2av1.Part_Text{Text: "ping"}}},
	}
	resp, err := handler.SendMessage(ctx, &a2av1.SendMessageRequest{Request: msg})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	task := resp.GetTask()
	if task == nil {
		t.Fatalf("expected task response")
	}
	approvalID := ""
	if metadata := task.GetStatus().GetMessage().GetMetadata(); metadata != nil {
		if value, ok := metadata.Fields["approval_id"]; ok {
			approvalID = value.GetStringValue()
		}
	}
	if approvalID == "" {
		t.Fatalf("expected approval_id metadata")
	}
	expired, err := handler.ExpireApprovals(ctx)
	if err != nil {
		t.Fatalf("expire approvals: %v", err)
	}
	if expired != 1 {
		t.Fatalf("expected 1 expired approval, got %d", expired)
	}
	finalTask, err := handler.Store.GetTask(ctx, task.Id, 0, true)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if finalTask.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_REJECTED {
		t.Fatalf("expected rejected, got %s", finalTask.GetStatus().GetState())
	}
	lateTask, err := handler.Approve(ctx, approvalID, "late")
	if err != nil {
		t.Fatalf("approve expired: %v", err)
	}
	if lateTask.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_REJECTED {
		t.Fatalf("expected rejected after expired approve")
	}
}
