package server

import (
	"context"
	"time"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Executor runs a task and returns a response message payload.
type Executor interface {
	Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error)
}

// SimpleHandler implements core A2A operations using a TaskStore and Executor.
type SimpleHandler struct {
	Store     TaskStore
	Executor  Executor
	AgentCard *a2av1.AgentCard
}

func (h *SimpleHandler) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	if h.Store == nil || h.Executor == nil {
		return nil, status.Error(codes.FailedPrecondition, "handler not configured")
	}
	message := req.GetRequest()
	if err := ValidateMessage(message); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	task, _, err := h.ensureTask(ctx, message)
	if err != nil {
		return nil, err
	}

	blocking := false
	if cfg := req.GetConfiguration(); cfg != nil {
		blocking = cfg.GetBlocking()
	}

	if blocking {
		respMsg, _, err := h.executeTask(ctx, task, message)
		if err != nil {
			return nil, err
		}
		return &a2av1.SendMessageResponse{Payload: &a2av1.SendMessageResponse_Msg{Msg: respMsg}}, nil
	}

	go h.runAsync(task.Id, message)

	return &a2av1.SendMessageResponse{Payload: &a2av1.SendMessageResponse_Task{Task: task}}, nil
}

func (h *SimpleHandler) SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
	if h.Store == nil || h.Executor == nil {
		return status.Error(codes.FailedPrecondition, "handler not configured")
	}
	message := req.GetRequest()
	if err := ValidateMessage(message); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	task, _, err := h.ensureTask(stream.Context(), message)
	if err != nil {
		return err
	}

	if err := stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_Task{Task: task}}); err != nil {
		return err
	}

	respMsg, artifacts, err := h.executeTask(stream.Context(), task, message)
	if err != nil {
		return err
	}

	if err := stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_Msg{Msg: respMsg}}); err != nil {
		return err
	}

	for _, artifact := range artifacts {
		event := &a2av1.TaskArtifactUpdateEvent{
			TaskId:    task.Id,
			ContextId: task.ContextId,
			Artifact:  artifact,
			Append:    true,
		}
		if err := stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_ArtifactUpdate{ArtifactUpdate: event}}); err != nil {
			return err
		}
	}

	statusEvent := &a2av1.TaskStatusUpdateEvent{
		TaskId:    task.Id,
		ContextId: task.ContextId,
		Status:    task.Status,
		Final:     true,
	}
	return stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_StatusUpdate{StatusUpdate: statusEvent}})
}

func (h *SimpleHandler) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	if h.Store == nil {
		return nil, status.Error(codes.FailedPrecondition, "task store not configured")
	}
	taskID := req.GetName()
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "task id is required")
	}

	task, err := h.Store.GetTask(ctx, taskID, req.GetHistoryLength(), false)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return task, nil
}

func (h *SimpleHandler) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	if h.Store == nil {
		return nil, status.Error(codes.FailedPrecondition, "task store not configured")
	}

	filter := TaskFilter{
		ContextID:        req.GetContextId(),
		Status:           req.GetStatus(),
		PageSize:         req.GetPageSize(),
		HistoryLength:    req.GetHistoryLength(),
		IncludeArtifacts: req.GetIncludeArtifacts(),
	}
	if req.GetLastUpdatedAfter() > 0 {
		filter.LastUpdatedAfter = time.UnixMilli(req.GetLastUpdatedAfter()).UTC()
	}

	tasks, total, err := h.Store.ListTasks(ctx, filter)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	return &a2av1.ListTasksResponse{
		Tasks:     tasks,
		PageSize:  pageSize,
		TotalSize: int32(total),
	}, nil
}

func (h *SimpleHandler) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	if h.Store == nil {
		return nil, status.Error(codes.FailedPrecondition, "task store not configured")
	}
	taskID := req.GetName()
	if taskID == "" {
		return nil, status.Error(codes.InvalidArgument, "task id is required")
	}
	task, err := h.Store.CancelTask(ctx, taskID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return task, nil
}

func (h *SimpleHandler) SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error {
	return status.Error(codes.Unimplemented, "SubscribeToTask not implemented")
}

func (h *SimpleHandler) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	if h.AgentCard == nil || !h.AgentCard.GetSupportsExtendedAgentCard() {
		return nil, status.Error(codes.Unimplemented, "extended agent card not supported")
	}
	if len(h.AgentCard.GetSkills()) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "extended agent card not configured")
	}
	return h.AgentCard, nil
}

func (h *SimpleHandler) ensureTask(ctx context.Context, message *a2av1.Message) (*a2av1.Task, bool, error) {
	if message.TaskId == "" {
		task, err := h.Store.CreateTask(ctx, message)
		if err != nil {
			return nil, false, status.Error(codes.Internal, err.Error())
		}
		return task, false, nil
	}

	task, err := h.Store.GetTask(ctx, message.TaskId, 0, true)
	if err != nil {
		return nil, false, status.Error(codes.NotFound, err.Error())
	}
	if isTerminalState(task.GetStatus().GetState()) {
		return nil, false, status.Error(codes.FailedPrecondition, "task is in terminal state")
	}
	message.ContextId = task.ContextId
	if err := h.Store.AppendHistory(ctx, task.Id, message); err != nil {
		return nil, false, status.Error(codes.Internal, err.Error())
	}
	return task, true, nil
}

func (h *SimpleHandler) executeTask(ctx context.Context, task *a2av1.Task, message *a2av1.Message) (*a2av1.Message, []*a2av1.Artifact, error) {
	statusWorking := newStatus(a2av1.TaskState_TASK_STATE_WORKING, message)
	_ = h.Store.UpdateStatus(ctx, task.Id, statusWorking)

	output, artifacts, err := h.Executor.Run(ctx, message)
	if err != nil {
		statusFailed := newStatus(a2av1.TaskState_TASK_STATE_FAILED, message)
		_ = h.Store.UpdateStatus(ctx, task.Id, statusFailed)
		return nil, nil, status.Error(codes.Internal, err.Error())
	}

	respMsg := ResponseMessage(output, task.ContextId, task.Id)
	_ = h.Store.AppendHistory(ctx, task.Id, respMsg)
	if len(artifacts) > 0 {
		_ = h.Store.AddArtifacts(ctx, task.Id, artifacts)
	}

	statusCompleted := newStatus(a2av1.TaskState_TASK_STATE_COMPLETED, respMsg)
	_ = h.Store.UpdateStatus(ctx, task.Id, statusCompleted)

	task.Status = statusCompleted
	return respMsg, artifacts, nil
}

func (h *SimpleHandler) runAsync(taskID string, message *a2av1.Message) {
	ctx := context.Background()
	task, err := h.Store.GetTask(ctx, taskID, 0, true)
	if err != nil {
		return
	}
	_, _, _ = h.executeTask(ctx, task, message)
}

func isTerminalState(state a2av1.TaskState) bool {
	switch state {
	case a2av1.TaskState_TASK_STATE_COMPLETED,
		a2av1.TaskState_TASK_STATE_FAILED,
		a2av1.TaskState_TASK_STATE_CANCELLED,
		a2av1.TaskState_TASK_STATE_REJECTED:
		return true
	default:
		return false
	}
}
