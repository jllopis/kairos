// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package client

import "google.golang.org/grpc"

// cancelOnFinishStream wraps a gRPC server-streaming client so that a
// context.CancelFunc is called when the stream finishes (either via Recv
// returning an error or the caller closing it). This prevents the timeout
// timer from leaking when the caller consumes the stream normally.
type cancelOnFinishStream[T any] struct {
	grpc.ServerStreamingClient[T]
	cancel func()
}

func (s *cancelOnFinishStream[T]) Recv() (*T, error) {
	msg, err := s.ServerStreamingClient.Recv()
	if err != nil {
		s.cancel()
	}
	return msg, err
}
