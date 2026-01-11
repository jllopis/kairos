package client

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

type testServer struct {
	a2av1.UnimplementedA2AServiceServer
	failFor     int32
	sleep       time.Duration
	streamSleep time.Duration
	attempts    int32
}

func (s *testServer) SendMessage(ctx context.Context, req *a2av1.SendMessageRequest) (*a2av1.SendMessageResponse, error) {
	attempt := atomic.AddInt32(&s.attempts, 1)
	if attempt <= s.failFor {
		return nil, status.Error(codes.Unavailable, "try again")
	}
	if s.sleep > 0 {
		select {
		case <-time.After(s.sleep):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return &a2av1.SendMessageResponse{}, nil
}

func (s *testServer) SendStreamingMessage(req *a2av1.SendMessageRequest, stream a2av1.A2AService_SendStreamingMessageServer) error {
	if s.streamSleep > 0 {
		select {
		case <-time.After(s.streamSleep):
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
	return stream.Send(&a2av1.StreamResponse{})
}

func (s *testServer) SubscribeToTask(req *a2av1.SubscribeToTaskRequest, stream a2av1.A2AService_SubscribeToTaskServer) error {
	if s.streamSleep > 0 {
		select {
		case <-time.After(s.streamSleep):
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
	}
	return stream.Send(&a2av1.StreamResponse{})
}

func newTestClient(t *testing.T, server *testServer) (grpc.ClientConnInterface, func()) {
	t.Helper()

	listener := bufconn.Listen(bufSize)
	grpcServer := grpc.NewServer()
	a2av1.RegisterA2AServiceServer(grpcServer, server)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	dialer := func(ctx context.Context, _ string) (net.Conn, error) {
		return listener.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(dialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("DialContext error: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}
	return conn, cleanup
}

func TestClientRetries_SucceedsAfterFailures(t *testing.T) {
	server := &testServer{failFor: 2}
	conn, cleanup := newTestClient(t, server)
	defer cleanup()

	client := New(conn, WithRetries(2))
	_, err := client.SendMessage(context.Background(), &a2av1.SendMessageRequest{})
	if err != nil {
		t.Fatalf("SendMessage error: %v", err)
	}
	if got := atomic.LoadInt32(&server.attempts); got != 3 {
		t.Fatalf("expected 3 attempts, got %d", got)
	}
}

func TestClientRetries_Exhausted(t *testing.T) {
	server := &testServer{failFor: 3}
	conn, cleanup := newTestClient(t, server)
	defer cleanup()

	client := New(conn, WithRetries(1))
	_, err := client.SendMessage(context.Background(), &a2av1.SendMessageRequest{})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", status.Code(err))
	}
	if got := atomic.LoadInt32(&server.attempts); got != 2 {
		t.Fatalf("expected 2 attempts, got %d", got)
	}
}

func TestClientTimeout(t *testing.T) {
	server := &testServer{sleep: 200 * time.Millisecond}
	conn, cleanup := newTestClient(t, server)
	defer cleanup()

	client := New(conn, WithTimeout(50*time.Millisecond))
	_, err := client.SendMessage(context.Background(), &a2av1.SendMessageRequest{})
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", status.Code(err))
	}
}

func TestClientStreamingTimeout(t *testing.T) {
	server := &testServer{streamSleep: 200 * time.Millisecond}
	conn, cleanup := newTestClient(t, server)
	defer cleanup()

	client := New(conn, WithTimeout(50*time.Millisecond))
	stream, err := client.SendStreamingMessage(context.Background(), &a2av1.SendMessageRequest{})
	if err != nil {
		t.Fatalf("SendStreamingMessage error: %v", err)
	}
	if _, err := stream.Recv(); status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", status.Code(err))
	}
}

func TestClientSubscribeTimeout(t *testing.T) {
	server := &testServer{streamSleep: 200 * time.Millisecond}
	conn, cleanup := newTestClient(t, server)
	defer cleanup()

	client := New(conn, WithTimeout(50*time.Millisecond))
	stream, err := client.SubscribeToTask(context.Background(), &a2av1.SubscribeToTaskRequest{Name: "tasks/abc"})
	if err != nil {
		t.Fatalf("SubscribeToTask error: %v", err)
	}
	if _, err := stream.Recv(); status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("expected DeadlineExceeded, got %v", status.Code(err))
	}
}

func TestClientStreamingSuccess(t *testing.T) {
	server := &testServer{streamSleep: 10 * time.Millisecond}
	conn, cleanup := newTestClient(t, server)
	defer cleanup()

	client := New(conn, WithTimeout(200*time.Millisecond))
	stream, err := client.SendStreamingMessage(context.Background(), &a2av1.SendMessageRequest{})
	if err != nil {
		t.Fatalf("SendStreamingMessage error: %v", err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("expected stream response, got %v", err)
	}
}

func TestClientSubscribeSuccess(t *testing.T) {
	server := &testServer{streamSleep: 10 * time.Millisecond}
	conn, cleanup := newTestClient(t, server)
	defer cleanup()

	client := New(conn, WithTimeout(200*time.Millisecond))
	stream, err := client.SubscribeToTask(context.Background(), &a2av1.SubscribeToTaskRequest{Name: "tasks/abc"})
	if err != nil {
		t.Fatalf("SubscribeToTask error: %v", err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatalf("expected stream response, got %v", err)
	}
}
