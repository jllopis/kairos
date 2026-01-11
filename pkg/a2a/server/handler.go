package server

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Executor runs a task and returns a response message payload.
type Executor interface {
	Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error)
}

// SimpleHandler implements core A2A operations using a TaskStore and Executor.
type SimpleHandler struct {
	Store    TaskStore
	Executor Executor
	Card     *a2av1.AgentCard
	PushCfgs PushConfigStore
}

// AgentCard exposes the configured agent card for capability checks.
func (h *SimpleHandler) AgentCard() *a2av1.AgentCard {
	return h.Card
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
		PageToken:        req.GetPageToken(),
		HistoryLength:    req.GetHistoryLength(),
		IncludeArtifacts: req.GetIncludeArtifacts(),
	}
	if req.GetLastUpdatedAfter() > 0 {
		filter.LastUpdatedAfter = time.UnixMilli(req.GetLastUpdatedAfter()).UTC()
	}

	tasks, total, err := h.Store.ListTasks(ctx, filter)
	if err != nil {
		if errors.Is(err, errInvalidPageToken) {
			return nil, status.Error(codes.InvalidArgument, "invalid page token")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}

	nextPageToken := ""
	offset, err := parsePageToken(filter.PageToken)
	if err == nil && offset+int(pageSize) < total {
		nextPageToken = strconv.Itoa(offset + int(pageSize))
	}

	return &a2av1.ListTasksResponse{
		Tasks:         tasks,
		PageSize:      pageSize,
		TotalSize:     int32(total),
		NextPageToken: nextPageToken,
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
	task, err := h.Store.GetTask(ctx, taskID, 0, true)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	state := task.GetStatus().GetState()
	if isTerminalState(state) && state != a2av1.TaskState_TASK_STATE_CANCELLED {
		return task, nil
	}
	task, err = h.Store.CancelTask(ctx, taskID)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return task, nil
}

func (h *SimpleHandler) SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error {
	if h.Store == nil {
		return status.Error(codes.FailedPrecondition, "task store not configured")
	}
	taskID, err := parseTaskName(req.GetName())
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	task, err := h.Store.GetTask(stream.Context(), taskID, 0, true)
	if err != nil {
		return status.Error(codes.NotFound, err.Error())
	}

	lastStatus := task.GetStatus()
	lastArtifactCount := len(task.GetArtifacts())

	if err := sendStatusUpdate(stream, task, lastStatus, isTerminalState(lastStatus.GetState())); err != nil {
		return err
	}
	if isTerminalState(lastStatus.GetState()) {
		return nil
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			latest, err := h.Store.GetTask(stream.Context(), taskID, 0, true)
			if err != nil {
				return status.Error(codes.NotFound, err.Error())
			}

			latestStatus := latest.GetStatus()
			statusChanged := !proto.Equal(lastStatus, latestStatus)
			if statusChanged {
				lastStatus = latestStatus
				final := isTerminalState(latestStatus.GetState())
				if err := sendStatusUpdate(stream, latest, latestStatus, final); err != nil {
					return err
				}
				if final {
					return nil
				}
			}

			artifactCount := len(latest.GetArtifacts())
			if artifactCount > lastArtifactCount {
				for _, artifact := range latest.GetArtifacts()[lastArtifactCount:] {
					event := &a2av1.TaskArtifactUpdateEvent{
						TaskId:    latest.Id,
						ContextId: latest.ContextId,
						Artifact:  artifact,
						Append:    true,
					}
					if err := stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_ArtifactUpdate{ArtifactUpdate: event}}); err != nil {
						return err
					}
				}
				lastArtifactCount = artifactCount
			}
		}
	}
}

func (h *SimpleHandler) SetTaskPushNotificationConfig(ctx context.Context, req *a2av1.SetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	if h.Store == nil || h.PushCfgs == nil {
		return nil, status.Error(codes.FailedPrecondition, "push config store not configured")
	}
	taskID, err := parseTaskName(req.GetParent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if _, err := h.Store.GetTask(ctx, taskID, 0, false); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	cfg := req.GetConfig()
	if cfg == nil || cfg.GetPushNotificationConfig() == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}
	if cfg.GetName() != "" {
		parsedTask, parsedConfig, err := parsePushConfigName(cfg.GetName())
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		if parsedTask != taskID {
			return nil, status.Error(codes.InvalidArgument, "config task mismatch")
		}
		if req.GetConfigId() == "" {
			req.ConfigId = parsedConfig
		} else if req.GetConfigId() != parsedConfig {
			return nil, status.Error(codes.InvalidArgument, "config id mismatch")
		}
	}

	configID := req.GetConfigId()
	pushCfg := cfg.GetPushNotificationConfig()
	if configID == "" {
		configID = pushCfg.GetId()
	}
	if configID == "" {
		configID = uuid.NewString()
	}
	if pushCfg.GetId() != "" && pushCfg.GetId() != configID {
		return nil, status.Error(codes.InvalidArgument, "config id mismatch")
	}
	cloned := proto.Clone(pushCfg).(*a2av1.PushNotificationConfig)
	cloned.Id = configID

	resource := &a2av1.TaskPushNotificationConfig{
		Name:                   pushConfigResourceName(taskID, configID),
		PushNotificationConfig: cloned,
	}
	stored, err := h.PushCfgs.Set(ctx, taskID, configID, resource)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return stored, nil
}

func (h *SimpleHandler) GetTaskPushNotificationConfig(ctx context.Context, req *a2av1.GetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	if h.PushCfgs == nil {
		return nil, status.Error(codes.FailedPrecondition, "push config store not configured")
	}
	taskID, configID, err := parsePushConfigName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	cfg, err := h.PushCfgs.Get(ctx, taskID, configID)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return cfg, nil
}

func (h *SimpleHandler) ListTaskPushNotificationConfig(ctx context.Context, req *a2av1.ListTaskPushNotificationConfigRequest) (*a2av1.ListTaskPushNotificationConfigResponse, error) {
	if h.PushCfgs == nil {
		return nil, status.Error(codes.FailedPrecondition, "push config store not configured")
	}
	if req.GetPageToken() != "" {
		return nil, status.Error(codes.InvalidArgument, "page tokens not supported")
	}
	taskID, err := parseTaskName(req.GetParent())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	configs, err := h.PushCfgs.List(ctx, taskID, req.GetPageSize())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &a2av1.ListTaskPushNotificationConfigResponse{
		Configs:       configs,
		NextPageToken: "",
	}, nil
}

func (h *SimpleHandler) DeleteTaskPushNotificationConfig(ctx context.Context, req *a2av1.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	if h.PushCfgs == nil {
		return nil, status.Error(codes.FailedPrecondition, "push config store not configured")
	}
	taskID, configID, err := parsePushConfigName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.PushCfgs.Delete(ctx, taskID, configID); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &emptypb.Empty{}, nil
}

func sendStatusUpdate(stream a2av1.A2AService_SubscribeToTaskServer, task *a2av1.Task, status *a2av1.TaskStatus, final bool) error {
	event := &a2av1.TaskStatusUpdateEvent{
		TaskId:    task.Id,
		ContextId: task.ContextId,
		Status:    status,
		Final:     final,
	}
	return stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_StatusUpdate{StatusUpdate: event}})
}

func (h *SimpleHandler) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	if h.Card == nil || !h.Card.GetSupportsExtendedAgentCard() {
		return nil, status.Error(codes.Unimplemented, "extended agent card not supported")
	}
	if len(h.Card.GetSkills()) == 0 {
		return nil, status.Error(codes.FailedPrecondition, "extended agent card not configured")
	}
	return h.Card, nil
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
