package server

import (
	"context"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Handler defines the core A2A operations for the gRPC binding.
type Handler interface {
	SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error)
	SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error
	GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error)
	ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error)
	CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error)
	SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error
	GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error)
}

// Service implements the A2A gRPC server by delegating to a Handler.
type Service struct {
	a2av1.UnimplementedA2AServiceServer
	handler Handler
}

// New creates a new Service instance.
func New(handler Handler) *Service {
	return &Service{handler: handler}
}

func (s *Service) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "SendMessage handler not configured")
	}
	return s.handler.SendMessage(ctx, req)
}

func (s *Service) SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
	if s.handler == nil {
		return status.Error(codes.Unimplemented, "SendStreamingMessage handler not configured")
	}
	return s.handler.SendStreamingMessage(req, stream)
}

func (s *Service) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "GetTask handler not configured")
	}
	return s.handler.GetTask(ctx, req)
}

func (s *Service) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "ListTasks handler not configured")
	}
	return s.handler.ListTasks(ctx, req)
}

func (s *Service) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "CancelTask handler not configured")
	}
	return s.handler.CancelTask(ctx, req)
}

func (s *Service) SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error {
	if s.handler == nil {
		return status.Error(codes.Unimplemented, "SubscribeToTask handler not configured")
	}
	return s.handler.SubscribeToTask(req, stream)
}

func (s *Service) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "GetExtendedAgentCard handler not configured")
	}
	return s.handler.GetExtendedAgentCard(ctx, req)
}

func (s *Service) SetTaskPushNotificationConfig(context.Context, *a2av1.SetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "SetTaskPushNotificationConfig not implemented")
}

func (s *Service) GetTaskPushNotificationConfig(context.Context, *a2av1.GetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	return nil, status.Error(codes.Unimplemented, "GetTaskPushNotificationConfig not implemented")
}

func (s *Service) ListTaskPushNotificationConfig(context.Context, *a2av1.ListTaskPushNotificationConfigRequest) (*a2av1.ListTaskPushNotificationConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListTaskPushNotificationConfig not implemented")
}

func (s *Service) DeleteTaskPushNotificationConfig(context.Context, *a2av1.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "DeleteTaskPushNotificationConfig not implemented")
}
