package client

import (
	"context"
	"time"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Option configures the A2A client.
type Option func(*Client)

// Client wraps the generated A2A gRPC client.
type Client struct {
	raw     a2av1.A2AServiceClient
	timeout time.Duration
	retries int
}

// New creates a client from an existing gRPC connection.
func New(conn grpc.ClientConnInterface, opts ...Option) *Client {
	client := &Client{
		raw:     a2av1.NewA2AServiceClient(conn),
		timeout: 10 * time.Second,
		retries: 0,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(client)
		}
	}
	return client
}

// WithTimeout sets a per-request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithRetries sets the number of retries for unary calls.
func WithRetries(retries int) Option {
	return func(c *Client) {
		if retries >= 0 {
			c.retries = retries
		}
	}
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

func (c *Client) streamContext(ctx context.Context) context.Context {
	if c.timeout <= 0 {
		return ctx
	}
	streamCtx, _ := context.WithTimeout(ctx, c.timeout)
	return streamCtx
}

// WithCredentials attaches per-RPC credentials to outgoing calls.
func WithCredentials(creds credentials.PerRPCCredentials) Option {
	return func(c *Client) {
		if creds == nil {
			return
		}
		c.raw = &authClient{inner: c.raw, creds: creds}
	}
}

type authClient struct {
	inner a2av1.A2AServiceClient
	creds credentials.PerRPCCredentials
}

func (a *authClient) SendMessage(ctx context.Context, in *a2av1.SendMessageRequest, opts ...grpc.CallOption) (*a2av1.SendMessageResponse, error) {
	return a.inner.SendMessage(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) SendStreamingMessage(ctx context.Context, in *a2av1.SendMessageRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[a2av1.StreamResponse], error) {
	return a.inner.SendStreamingMessage(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) GetTask(ctx context.Context, in *a2av1.GetTaskRequest, opts ...grpc.CallOption) (*a2av1.Task, error) {
	return a.inner.GetTask(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) ListTasks(ctx context.Context, in *a2av1.ListTasksRequest, opts ...grpc.CallOption) (*a2av1.ListTasksResponse, error) {
	return a.inner.ListTasks(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) CancelTask(ctx context.Context, in *a2av1.CancelTaskRequest, opts ...grpc.CallOption) (*a2av1.Task, error) {
	return a.inner.CancelTask(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) SubscribeToTask(ctx context.Context, in *a2av1.SubscribeToTaskRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[a2av1.StreamResponse], error) {
	return a.inner.SubscribeToTask(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) SetTaskPushNotificationConfig(ctx context.Context, in *a2av1.SetTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*a2av1.TaskPushNotificationConfig, error) {
	return a.inner.SetTaskPushNotificationConfig(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) GetTaskPushNotificationConfig(ctx context.Context, in *a2av1.GetTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*a2av1.TaskPushNotificationConfig, error) {
	return a.inner.GetTaskPushNotificationConfig(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) ListTaskPushNotificationConfig(ctx context.Context, in *a2av1.ListTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*a2av1.ListTaskPushNotificationConfigResponse, error) {
	return a.inner.ListTaskPushNotificationConfig(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) GetExtendedAgentCard(ctx context.Context, in *a2av1.GetExtendedAgentCardRequest, opts ...grpc.CallOption) (*a2av1.AgentCard, error) {
	return a.inner.GetExtendedAgentCard(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

func (a *authClient) DeleteTaskPushNotificationConfig(ctx context.Context, in *a2av1.DeleteTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return a.inner.DeleteTaskPushNotificationConfig(ctx, in, append(opts, grpc.PerRPCCredentials(a.creds))...)
}

// SendMessage forwards to the A2A SendMessage RPC.
func (c *Client) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest, opts ...grpc.CallOption) (*a2av1.SendMessageResponse, error) {
	return withRetries(c.retries, func() (*a2av1.SendMessageResponse, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.SendMessage(ctx, req, opts...)
	})
}

// SendStreamingMessage forwards to the A2A SendStreamingMessage RPC.
func (c *Client) SendStreamingMessage(ctx context.Context, req *a2av1.SendMessageRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[a2av1.StreamResponse], error) {
	return c.raw.SendStreamingMessage(c.streamContext(ctx), req, opts...)
}

// GetTask forwards to the A2A GetTask RPC.
func (c *Client) GetTask(ctx context.Context, req *a2av1.GetTaskRequest, opts ...grpc.CallOption) (*a2av1.Task, error) {
	return withRetries(c.retries, func() (*a2av1.Task, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.GetTask(ctx, req, opts...)
	})
}

// ListTasks forwards to the A2A ListTasks RPC.
func (c *Client) ListTasks(ctx context.Context, req *a2av1.ListTasksRequest, opts ...grpc.CallOption) (*a2av1.ListTasksResponse, error) {
	return withRetries(c.retries, func() (*a2av1.ListTasksResponse, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.ListTasks(ctx, req, opts...)
	})
}

// CancelTask forwards to the A2A CancelTask RPC.
func (c *Client) CancelTask(ctx context.Context, req *a2av1.CancelTaskRequest, opts ...grpc.CallOption) (*a2av1.Task, error) {
	return withRetries(c.retries, func() (*a2av1.Task, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.CancelTask(ctx, req, opts...)
	})
}

// SubscribeToTask forwards to the A2A SubscribeToTask RPC.
func (c *Client) SubscribeToTask(ctx context.Context, req *a2av1.SubscribeToTaskRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[a2av1.StreamResponse], error) {
	return c.raw.SubscribeToTask(c.streamContext(ctx), req, opts...)
}

// GetExtendedAgentCard forwards to the A2A GetExtendedAgentCard RPC.
func (c *Client) GetExtendedAgentCard(ctx context.Context, req *a2av1.GetExtendedAgentCardRequest, opts ...grpc.CallOption) (*a2av1.AgentCard, error) {
	return withRetries(c.retries, func() (*a2av1.AgentCard, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.GetExtendedAgentCard(ctx, req, opts...)
	})
}

// SetTaskPushNotificationConfig forwards to the A2A SetTaskPushNotificationConfig RPC.
func (c *Client) SetTaskPushNotificationConfig(ctx context.Context, req *a2av1.SetTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*a2av1.TaskPushNotificationConfig, error) {
	return withRetries(c.retries, func() (*a2av1.TaskPushNotificationConfig, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.SetTaskPushNotificationConfig(ctx, req, opts...)
	})
}

// GetTaskPushNotificationConfig forwards to the A2A GetTaskPushNotificationConfig RPC.
func (c *Client) GetTaskPushNotificationConfig(ctx context.Context, req *a2av1.GetTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*a2av1.TaskPushNotificationConfig, error) {
	return withRetries(c.retries, func() (*a2av1.TaskPushNotificationConfig, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.GetTaskPushNotificationConfig(ctx, req, opts...)
	})
}

// ListTaskPushNotificationConfig forwards to the A2A ListTaskPushNotificationConfig RPC.
func (c *Client) ListTaskPushNotificationConfig(ctx context.Context, req *a2av1.ListTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*a2av1.ListTaskPushNotificationConfigResponse, error) {
	return withRetries(c.retries, func() (*a2av1.ListTaskPushNotificationConfigResponse, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.ListTaskPushNotificationConfig(ctx, req, opts...)
	})
}

// DeleteTaskPushNotificationConfig forwards to the A2A DeleteTaskPushNotificationConfig RPC.
func (c *Client) DeleteTaskPushNotificationConfig(ctx context.Context, req *a2av1.DeleteTaskPushNotificationConfigRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	return withRetries(c.retries, func() (*emptypb.Empty, error) {
		ctx, cancel := c.withTimeout(ctx)
		defer cancel()
		return c.raw.DeleteTaskPushNotificationConfig(ctx, req, opts...)
	})
}

func withRetries[T any](retries int, fn func() (*T, error)) (*T, error) {
	var lastErr error
	for attempt := 0; attempt <= retries; attempt++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
