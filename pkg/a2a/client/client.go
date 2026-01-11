package client

import (
	"context"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc"
)

// Client wraps the generated A2A gRPC client.
type Client struct {
	raw a2av1.A2AServiceClient
}

// New creates a client from an existing gRPC connection.
func New(conn grpc.ClientConnInterface) *Client {
	return &Client{raw: a2av1.NewA2AServiceClient(conn)}
}

// SendMessage forwards to the A2A SendMessage RPC.
func (c *Client) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest, opts ...grpc.CallOption) (*a2av1.SendMessageResponse, error) {
	return c.raw.SendMessage(ctx, req, opts...)
}

// SendStreamingMessage forwards to the A2A SendStreamingMessage RPC.
func (c *Client) SendStreamingMessage(ctx context.Context, req *a2av1.SendMessageRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[a2av1.StreamResponse], error) {
	return c.raw.SendStreamingMessage(ctx, req, opts...)
}

// GetTask forwards to the A2A GetTask RPC.
func (c *Client) GetTask(ctx context.Context, req *a2av1.GetTaskRequest, opts ...grpc.CallOption) (*a2av1.Task, error) {
	return c.raw.GetTask(ctx, req, opts...)
}

// ListTasks forwards to the A2A ListTasks RPC.
func (c *Client) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest, opts ...grpc.CallOption) (*a2av1.ListTasksResponse, error) {
	return c.raw.ListTasks(ctx, req, opts...)
}

// CancelTask forwards to the A2A CancelTask RPC.
func (c *Client) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest, opts ...grpc.CallOption) (*a2av1.Task, error) {
	return c.raw.CancelTask(ctx, req, opts...)
}

// SubscribeToTask forwards to the A2A SubscribeToTask RPC.
func (c *Client) SubscribeToTask(ctx context.Context, req *a2av1.SubscribeToTaskRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[a2av1.StreamResponse], error) {
	return c.raw.SubscribeToTask(ctx, req, opts...)
}

// GetExtendedAgentCard forwards to the A2A GetExtendedAgentCard RPC.
func (c *Client) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest, opts ...grpc.CallOption) (*a2av1.AgentCard, error) {
	return c.raw.GetExtendedAgentCard(ctx, req, opts...)
}
