// Package server implements the A2A gRPC server binding and core handlers.
package server

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"github.com/jllopis/kairos/pkg/governance"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

// Executor runs a task and returns a response message payload.
type Executor interface {
	Run(ctx context.Context, message *a2av1.Message) (any, []*a2av1.Artifact, error)
}

// SimpleHandler implements core A2A operations using a TaskStore and Executor.
type SimpleHandler struct {
	Store           TaskStore
	Executor        Executor
	Card            *a2av1.AgentCard
	PushCfgs        PushConfigStore
	PolicyEngine    governance.PolicyEngine
	ApprovalHook    governance.ApprovalHook
	ApprovalStore   ApprovalStore
	ApprovalTimeout time.Duration
}

// AgentCard exposes the configured agent card for capability checks.
func (h *SimpleHandler) AgentCard() *a2av1.AgentCard {
	return h.Card
}

// SendMessage creates or updates a task and optionally executes it synchronously.
func (h *SimpleHandler) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	if h.Store == nil || h.Executor == nil {
		return nil, status.Error(codes.FailedPrecondition, "handler not configured")
	}
	message := req.GetRequest()
	if err := ValidateMessage(message); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if response, ok, err := h.applyPolicy(ctx, message); ok {
		return response, err
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

// SendStreamingMessage executes a task and streams task/message/artifact updates.
func (h *SimpleHandler) SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
	if h.Store == nil || h.Executor == nil {
		return status.Error(codes.FailedPrecondition, "handler not configured")
	}
	message := req.GetRequest()
	if err := ValidateMessage(message); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	handled, err := h.applyStreamingPolicy(stream.Context(), message, stream)
	if err != nil {
		return err
	}
	if handled {
		return nil
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

// GetTask retrieves a task by name.
func (h *SimpleHandler) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	if h.Store == nil {
		return nil, status.Error(codes.FailedPrecondition, "task store not configured")
	}
	taskID, err := parseTaskName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if req.GetHistoryLength() < 0 {
		return nil, status.Error(codes.InvalidArgument, "history length must be >= 0")
	}

	task, err := h.Store.GetTask(ctx, taskID, req.GetHistoryLength(), false)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return task, nil
}

// ListTasks lists tasks using request filters and pagination.
func (h *SimpleHandler) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	if h.Store == nil {
		return nil, status.Error(codes.FailedPrecondition, "task store not configured")
	}
	if req.GetHistoryLength() < 0 {
		return nil, status.Error(codes.InvalidArgument, "history length must be >= 0")
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
	if pageSize < 0 {
		return nil, status.Error(codes.InvalidArgument, "page size must be >= 0")
	}
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

// CancelTask cancels a task if it is not terminal.
func (h *SimpleHandler) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	if h.Store == nil {
		return nil, status.Error(codes.FailedPrecondition, "task store not configured")
	}
	taskID, err := parseTaskName(req.GetName())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
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

// SubscribeToTask streams status and artifact updates for a task.
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

// SetTaskPushNotificationConfig creates or updates a task push config.
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

// GetTaskPushNotificationConfig retrieves a task push config.
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

// ListTaskPushNotificationConfig lists push configs for a task.
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

// DeleteTaskPushNotificationConfig deletes a task push config.
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

// GetExtendedAgentCard returns the extended AgentCard if available.
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

func (h *SimpleHandler) applyPolicy(ctx context.Context, message *a2av1.Message) (*a2av1.SendMessageResponse, bool, error) {
	if h.PolicyEngine == nil {
		return nil, false, nil
	}
	action := h.policyAction(message)
	decision := h.PolicyEngine.Evaluate(ctx, action)
	if decision.IsPending() && h.ApprovalHook != nil {
		decision = h.ApprovalHook.Request(ctx, action)
	}
	if decision.IsAllowed() {
		return nil, false, nil
	}
	task, _, err := h.ensureTask(ctx, message)
	if err != nil {
		return nil, true, err
	}
	message.TaskId = task.Id
	message.ContextId = task.ContextId
	approvalID := ""
	expiresAt := time.Time{}
	if decision.IsPending() && h.ApprovalStore != nil {
		expiresAt = approvalExpiry(h.ApprovalTimeout)
		if record, err := h.ApprovalStore.Create(ctx, ApprovalRecord{
			TaskID:    task.Id,
			ContextID: task.ContextId,
			Status:    ApprovalStatusPending,
			Reason:    decision.Reason,
			ExpiresAt: expiresAt,
			Message:   message,
		}); err == nil {
			approvalID = record.ID
		}
	}
	state, reason := policyStatus(decision)
	statusMsg := ResponseMessage(reason, task.ContextId, task.Id)
	if approvalID != "" {
		metadata := map[string]string{"approval_id": approvalID}
		if !expiresAt.IsZero() {
			metadata["approval_expires_at"] = expiresAt.UTC().Format(time.RFC3339)
		}
		statusMsg.Metadata = mergeMetadata(statusMsg.Metadata, metadata)
	}
	_ = h.Store.AppendHistory(ctx, task.Id, statusMsg)
	status := newStatus(state, statusMsg)
	_ = h.Store.UpdateStatus(ctx, task.Id, status)
	task.Status = status
	return &a2av1.SendMessageResponse{Payload: &a2av1.SendMessageResponse_Task{Task: task}}, true, nil
}

func (h *SimpleHandler) applyStreamingPolicy(ctx context.Context, message *a2av1.Message, stream a2av1.A2AService_SendStreamingMessageServer) (bool, error) {
	if h.PolicyEngine == nil {
		return false, nil
	}
	action := h.policyAction(message)
	decision := h.PolicyEngine.Evaluate(ctx, action)
	if decision.IsPending() && h.ApprovalHook != nil {
		decision = h.ApprovalHook.Request(ctx, action)
	}
	if decision.IsAllowed() {
		return false, nil
	}
	task, _, err := h.ensureTask(ctx, message)
	if err != nil {
		return true, err
	}
	message.TaskId = task.Id
	message.ContextId = task.ContextId
	approvalID := ""
	expiresAt := time.Time{}
	if decision.IsPending() && h.ApprovalStore != nil {
		expiresAt = approvalExpiry(h.ApprovalTimeout)
		if record, err := h.ApprovalStore.Create(ctx, ApprovalRecord{
			TaskID:    task.Id,
			ContextID: task.ContextId,
			Status:    ApprovalStatusPending,
			Reason:    decision.Reason,
			ExpiresAt: expiresAt,
			Message:   message,
		}); err == nil {
			approvalID = record.ID
		}
	}
	if err := stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_Task{Task: task}}); err != nil {
		return true, err
	}
	state, reason := policyStatus(decision)
	statusMsg := ResponseMessage(reason, task.ContextId, task.Id)
	if approvalID != "" {
		metadata := map[string]string{"approval_id": approvalID}
		if !expiresAt.IsZero() {
			metadata["approval_expires_at"] = expiresAt.UTC().Format(time.RFC3339)
		}
		statusMsg.Metadata = mergeMetadata(statusMsg.Metadata, metadata)
	}
	_ = h.Store.AppendHistory(ctx, task.Id, statusMsg)
	status := newStatus(state, statusMsg)
	_ = h.Store.UpdateStatus(ctx, task.Id, status)
	task.Status = status
	statusEvent := &a2av1.TaskStatusUpdateEvent{
		TaskId:    task.Id,
		ContextId: task.ContextId,
		Status:    status,
		Final:     state == a2av1.TaskState_TASK_STATE_REJECTED,
	}
	return true, stream.Send(&a2av1.StreamResponse{Payload: &a2av1.StreamResponse_StatusUpdate{StatusUpdate: statusEvent}})
}

func (h *SimpleHandler) policyAction(message *a2av1.Message) governance.Action {
	name := "a2a-handler"
	if h.Card != nil && strings.TrimSpace(h.Card.Name) != "" {
		name = h.Card.Name
	}
	return governance.Action{
		Type:     governance.ActionAgent,
		Name:     name,
		Metadata: policyMetadata(message),
	}
}

func policyMetadata(message *a2av1.Message) map[string]string {
	if message == nil {
		return nil
	}
	meta := map[string]string{
		"message_id": message.GetMessageId(),
		"context_id": message.GetContextId(),
		"task_id":    message.GetTaskId(),
	}
	extra := message.GetMetadata()
	if extra == nil {
		return meta
	}
	for _, key := range []string{"caller", "agent", "tenant"} {
		if val, ok := extra.Fields[key]; ok {
			if str := val.GetStringValue(); str != "" {
				meta[key] = str
			}
		}
	}
	return meta
}

func policyStatus(decision governance.Decision) (a2av1.TaskState, string) {
	reason := strings.TrimSpace(decision.Reason)
	if decision.IsPending() {
		if reason == "" {
			reason = "approval required"
		}
		return a2av1.TaskState_TASK_STATE_INPUT_REQUIRED, reason
	}
	if reason == "" {
		reason = "blocked by policy"
	}
	return a2av1.TaskState_TASK_STATE_REJECTED, reason
}

// Approve resolves a pending approval and executes the task.
func (h *SimpleHandler) Approve(ctx context.Context, id, reason string) (*a2av1.Task, error) {
	if h.ApprovalStore == nil {
		return nil, status.Error(codes.FailedPrecondition, "approval store not configured")
	}
	approval, err := h.ApprovalStore.Get(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if approval.Status == ApprovalStatusApproved {
		return h.Store.GetTask(ctx, approval.TaskID, 0, true)
	}
	if approval.Status == ApprovalStatusRejected {
		return h.Store.GetTask(ctx, approval.TaskID, 0, true)
	}
	if isApprovalExpired(approval) {
		_, _ = h.ApprovalStore.UpdateStatus(ctx, id, ApprovalStatusRejected, "approval expired")
		return h.Reject(ctx, id, "approval expired")
	}
	if approval.Message == nil {
		return nil, status.Error(codes.FailedPrecondition, "approval has no message")
	}
	if _, err := h.ApprovalStore.UpdateStatus(ctx, id, ApprovalStatusApproved, reason); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	task, err := h.Store.GetTask(ctx, approval.TaskID, 0, true)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if isTerminalState(task.GetStatus().GetState()) {
		return task, nil
	}
	if _, _, err := h.executeTask(ctx, task, approval.Message); err != nil {
		return nil, err
	}
	return h.Store.GetTask(ctx, approval.TaskID, 0, true)
}

// Reject resolves a pending approval as rejected.
func (h *SimpleHandler) Reject(ctx context.Context, id, reason string) (*a2av1.Task, error) {
	if h.ApprovalStore == nil {
		return nil, status.Error(codes.FailedPrecondition, "approval store not configured")
	}
	approval, err := h.ApprovalStore.Get(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	if approval.Status == ApprovalStatusRejected {
		return h.Store.GetTask(ctx, approval.TaskID, 0, true)
	}
	if isApprovalExpired(approval) {
		_, _ = h.ApprovalStore.UpdateStatus(ctx, id, ApprovalStatusRejected, "approval expired")
	}
	if _, err := h.ApprovalStore.UpdateStatus(ctx, id, ApprovalStatusRejected, reason); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	task, err := h.Store.GetTask(ctx, approval.TaskID, 0, true)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	statusMsg := ResponseMessage(reason, task.ContextId, task.Id)
	statusMsg.Metadata = mergeMetadata(statusMsg.Metadata, map[string]string{
		"approval_id": id,
	})
	_ = h.Store.AppendHistory(ctx, task.Id, statusMsg)
	status := newStatus(a2av1.TaskState_TASK_STATE_REJECTED, statusMsg)
	_ = h.Store.UpdateStatus(ctx, task.Id, status)
	task.Status = status
	return task, nil
}

// GetApproval returns a single approval record.
func (h *SimpleHandler) GetApproval(ctx context.Context, id string) (*ApprovalRecord, error) {
	if h.ApprovalStore == nil {
		return nil, status.Error(codes.FailedPrecondition, "approval store not configured")
	}
	record, err := h.ApprovalStore.Get(ctx, id)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return record, nil
}

// ListApprovals returns approval records.
func (h *SimpleHandler) ListApprovals(ctx context.Context, filter ApprovalFilter) ([]*ApprovalRecord, error) {
	if h.ApprovalStore == nil {
		return nil, status.Error(codes.FailedPrecondition, "approval store not configured")
	}
	return h.ApprovalStore.List(ctx, filter)
}

func mergeMetadata(existing *structpb.Struct, values map[string]string) *structpb.Struct {
	if len(values) == 0 {
		return existing
	}
	out := map[string]interface{}{}
	if existing != nil {
		for key, value := range existing.AsMap() {
			out[key] = value
		}
	}
	for key, value := range values {
		out[key] = value
	}
	merged, err := structpb.NewStruct(out)
	if err != nil {
		return existing
	}
	return merged
}

// ExpireApprovals rejects pending approvals that passed their expiry.
func (h *SimpleHandler) ExpireApprovals(ctx context.Context) (int, error) {
	if h.ApprovalStore == nil {
		return 0, status.Error(codes.FailedPrecondition, "approval store not configured")
	}
	now := time.Now().UTC()
	records, err := h.ApprovalStore.List(ctx, ApprovalFilter{
		Status:         ApprovalStatusPending,
		ExpiringBefore: now,
	})
	if err != nil {
		return 0, status.Error(codes.Internal, err.Error())
	}
	expired := 0
	for _, record := range records {
		if record == nil {
			continue
		}
		if _, err := h.Reject(ctx, record.ID, "approval expired"); err == nil {
			expired++
		}
	}
	return expired, nil
}

func approvalExpiry(timeout time.Duration) time.Time {
	if timeout == 0 {
		return time.Time{}
	}
	if timeout < 0 {
		return time.Now().UTC().Add(timeout)
	}
	return time.Now().UTC().Add(timeout)
}

func isApprovalExpired(record *ApprovalRecord) bool {
	if record == nil {
		return false
	}
	if record.ExpiresAt.IsZero() {
		return false
	}
	return time.Now().UTC().After(record.ExpiresAt)
}
