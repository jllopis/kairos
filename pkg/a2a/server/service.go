package server

import (
	"context"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	tracer  trace.Tracer
}

// New creates a new Service instance.
func New(handler Handler) *Service {
	return &Service{
		handler: handler,
		tracer:  otel.Tracer("kairos/a2a"),
	}
}

// SendMessage handles the SendMessage RPC.
func (s *Service) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "SendMessage handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.SendMessage", trace.WithAttributes(
		attribute.String("a2a.method", "SendMessage"),
	))
	defer span.End()
	return s.handler.SendMessage(ctx, req)
}

// SendStreamingMessage handles the streaming SendMessage RPC.
func (s *Service) SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
	if s.handler == nil {
		return status.Error(codes.Unimplemented, "SendStreamingMessage handler not configured")
	}
	if !supportsStreaming(s.handler) {
		return status.Error(codes.Unimplemented, "streaming not supported")
	}
	ctx, span := s.tracer.Start(stream.Context(), "A2A.SendStreamingMessage", trace.WithAttributes(
		attribute.String("a2a.method", "SendStreamingMessage"),
		attribute.Bool("a2a.stream", true),
	))
	defer span.End()
	return s.handler.SendStreamingMessage(req, wrapStreamContext(stream, ctx))
}

// GetTask handles the GetTask RPC.
func (s *Service) GetTask(ctx context.Context, req *a2av1.GetTaskRequest) (*a2av1.Task, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "GetTask handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.GetTask", trace.WithAttributes(
		attribute.String("a2a.method", "GetTask"),
		attribute.String("a2a.task_id", req.GetName()),
	))
	defer span.End()
	return s.handler.GetTask(ctx, req)
}

// ListTasks handles the ListTasks RPC.
func (s *Service) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest) (*a2av1.ListTasksResponse, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "ListTasks handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.ListTasks", trace.WithAttributes(
		attribute.String("a2a.method", "ListTasks"),
		attribute.String("a2a.context_id", req.GetContextId()),
	))
	defer span.End()
	return s.handler.ListTasks(ctx, req)
}

// CancelTask handles the CancelTask RPC.
func (s *Service) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest) (*a2av1.Task, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "CancelTask handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.CancelTask", trace.WithAttributes(
		attribute.String("a2a.method", "CancelTask"),
		attribute.String("a2a.task_id", req.GetName()),
	))
	defer span.End()
	return s.handler.CancelTask(ctx, req)
}

// SubscribeToTask handles the SubscribeToTask streaming RPC.
func (s *Service) SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error {
	if s.handler == nil {
		return status.Error(codes.Unimplemented, "SubscribeToTask handler not configured")
	}
	if !supportsStreaming(s.handler) {
		return status.Error(codes.Unimplemented, "streaming not supported")
	}
	ctx, span := s.tracer.Start(stream.Context(), "A2A.SubscribeToTask", trace.WithAttributes(
		attribute.String("a2a.method", "SubscribeToTask"),
		attribute.Bool("a2a.stream", true),
		attribute.String("a2a.task_id", req.GetName()),
	))
	defer span.End()
	return s.handler.SubscribeToTask(req, wrapSubscribeContext(stream, ctx))
}

// GetExtendedAgentCard handles the GetExtendedAgentCard RPC.
func (s *Service) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest) (*a2av1.AgentCard, error) {
	if s.handler == nil {
		return nil, status.Error(codes.Unimplemented, "GetExtendedAgentCard handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.GetExtendedAgentCard", trace.WithAttributes(
		attribute.String("a2a.method", "GetExtendedAgentCard"),
	))
	defer span.End()
	return s.handler.GetExtendedAgentCard(ctx, req)
}

// SetTaskPushNotificationConfig handles the SetTaskPushNotificationConfig RPC.
func (s *Service) SetTaskPushNotificationConfig(ctx context.Context, req *a2av1.SetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	if !supportsPushNotifications(s.handler) {
		return nil, status.Error(codes.Unimplemented, "push notifications not supported")
	}
	handler, ok := s.handler.(pushNotificationHandler)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "SetTaskPushNotificationConfig handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.SetTaskPushNotificationConfig", trace.WithAttributes(
		attribute.String("a2a.method", "SetTaskPushNotificationConfig"),
		attribute.String("a2a.task_id", req.GetParent()),
	))
	defer span.End()
	return handler.SetTaskPushNotificationConfig(ctx, req)
}

// GetTaskPushNotificationConfig handles the GetTaskPushNotificationConfig RPC.
func (s *Service) GetTaskPushNotificationConfig(ctx context.Context, req *a2av1.GetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error) {
	if !supportsPushNotifications(s.handler) {
		return nil, status.Error(codes.Unimplemented, "push notifications not supported")
	}
	handler, ok := s.handler.(pushNotificationHandler)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "GetTaskPushNotificationConfig handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.GetTaskPushNotificationConfig", trace.WithAttributes(
		attribute.String("a2a.method", "GetTaskPushNotificationConfig"),
		attribute.String("a2a.config_name", req.GetName()),
	))
	defer span.End()
	return handler.GetTaskPushNotificationConfig(ctx, req)
}

// ListTaskPushNotificationConfig handles the ListTaskPushNotificationConfig RPC.
func (s *Service) ListTaskPushNotificationConfig(ctx context.Context, req *a2av1.ListTaskPushNotificationConfigRequest) (*a2av1.ListTaskPushNotificationConfigResponse, error) {
	if !supportsPushNotifications(s.handler) {
		return nil, status.Error(codes.Unimplemented, "push notifications not supported")
	}
	handler, ok := s.handler.(pushNotificationHandler)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "ListTaskPushNotificationConfig handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.ListTaskPushNotificationConfig", trace.WithAttributes(
		attribute.String("a2a.method", "ListTaskPushNotificationConfig"),
		attribute.String("a2a.task_id", req.GetParent()),
	))
	defer span.End()
	return handler.ListTaskPushNotificationConfig(ctx, req)
}

// DeleteTaskPushNotificationConfig handles the DeleteTaskPushNotificationConfig RPC.
func (s *Service) DeleteTaskPushNotificationConfig(ctx context.Context, req *a2av1.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error) {
	if !supportsPushNotifications(s.handler) {
		return nil, status.Error(codes.Unimplemented, "push notifications not supported")
	}
	handler, ok := s.handler.(pushNotificationHandler)
	if !ok {
		return nil, status.Error(codes.Unimplemented, "DeleteTaskPushNotificationConfig handler not configured")
	}
	ctx, span := s.tracer.Start(ctx, "A2A.DeleteTaskPushNotificationConfig", trace.WithAttributes(
		attribute.String("a2a.method", "DeleteTaskPushNotificationConfig"),
		attribute.String("a2a.config_name", req.GetName()),
	))
	defer span.End()
	return handler.DeleteTaskPushNotificationConfig(ctx, req)
}

type pushNotificationHandler interface {
	SetTaskPushNotificationConfig(ctx context.Context, req *a2av1.SetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error)
	GetTaskPushNotificationConfig(ctx context.Context, req *a2av1.GetTaskPushNotificationConfigRequest) (*a2av1.TaskPushNotificationConfig, error)
	ListTaskPushNotificationConfig(ctx context.Context, req *a2av1.ListTaskPushNotificationConfigRequest) (*a2av1.ListTaskPushNotificationConfigResponse, error)
	DeleteTaskPushNotificationConfig(ctx context.Context, req *a2av1.DeleteTaskPushNotificationConfigRequest) (*emptypb.Empty, error)
}

type agentCardProvider interface {
	AgentCard() *a2av1.AgentCard
}

func supportsStreaming(handler Handler) bool {
	card := readAgentCard(handler)
	if card == nil || card.Capabilities == nil {
		return false
	}
	return card.Capabilities.GetStreaming()
}

func supportsPushNotifications(handler Handler) bool {
	card := readAgentCard(handler)
	if card == nil || card.Capabilities == nil {
		return false
	}
	return card.Capabilities.GetPushNotifications()
}

func readAgentCard(handler Handler) *a2av1.AgentCard {
	if provider, ok := handler.(agentCardProvider); ok {
		return provider.AgentCard()
	}
	return nil
}

type sendMessageStreamWrapper struct {
	a2av1.A2AService_SendStreamingMessageServer
	ctx context.Context
}

func (s sendMessageStreamWrapper) Context() context.Context {
	return s.ctx
}

func wrapStreamContext(stream a2av1.A2AService_SendStreamingMessageServer, ctx context.Context) a2av1.A2AService_SendStreamingMessageServer {
	return sendMessageStreamWrapper{A2AService_SendStreamingMessageServer: stream, ctx: ctx}
}

type subscribeStreamWrapper struct {
	a2av1.A2AService_SubscribeToTaskServer
	ctx context.Context
}

func (s subscribeStreamWrapper) Context() context.Context {
	return s.ctx
}

func wrapSubscribeContext(stream a2av1.A2AService_SubscribeToTaskServer, ctx context.Context) a2av1.A2AService_SubscribeToTaskServer {
	return subscribeStreamWrapper{A2AService_SubscribeToTaskServer: stream, ctx: ctx}
}
