package server

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// AuthConfig defines minimal auth requirements for A2A handlers.
// This is a stub configuration that validates presence of auth signals only.
type AuthConfig struct {
	RequireOIDC bool
	RequireMTLS bool
}

// AuthContext holds request metadata for auth checks.
type AuthContext struct {
	Method   string
	Metadata metadata.MD
	Peer     *peer.Peer
}

// Authenticator validates a request and returns an error when unauthenticated.
type Authenticator interface {
	Authenticate(ctx context.Context, auth AuthContext) error
}

// NewAuthenticator builds an authenticator using a minimal presence check.
func NewAuthenticator(cfg AuthConfig) Authenticator {
	return &authenticator{cfg: cfg}
}

// UnaryAuthInterceptor returns a gRPC unary interceptor for A2A auth stubs.
func UnaryAuthInterceptor(auth Authenticator) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if auth != nil {
			if err := auth.Authenticate(ctx, authContext(ctx, info.FullMethod)); err != nil {
				return nil, status.Error(codes.Unauthenticated, err.Error())
			}
		}
		return handler(ctx, req)
	}
}

// StreamAuthInterceptor returns a gRPC stream interceptor for A2A auth stubs.
func StreamAuthInterceptor(auth Authenticator) grpc.StreamServerInterceptor {
	return func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if auth != nil {
			if err := auth.Authenticate(stream.Context(), authContext(stream.Context(), info.FullMethod)); err != nil {
				return status.Error(codes.Unauthenticated, err.Error())
			}
		}
		return handler(srv, stream)
	}
}

type authenticator struct {
	cfg AuthConfig
}

func (a *authenticator) Authenticate(ctx context.Context, auth AuthContext) error {
	if a.cfg.RequireOIDC {
		if token := bearerToken(auth.Metadata); token == "" {
			return errors.New("missing bearer token")
		}
	}
	if a.cfg.RequireMTLS {
		if !hasMTLS(auth.Peer) {
			return errors.New("mutual TLS required")
		}
	}
	return nil
}

func authContext(ctx context.Context, method string) AuthContext {
	md, _ := metadata.FromIncomingContext(ctx)
	p, _ := peer.FromContext(ctx)
	return AuthContext{
		Method:   method,
		Metadata: md,
		Peer:     p,
	}
}

func bearerToken(md metadata.MD) string {
	if md == nil {
		return ""
	}
	values := md.Get("authorization")
	if len(values) == 0 {
		return ""
	}
	value := values[0]
	if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return ""
	}
	return strings.TrimSpace(value[len("bearer "):])
}

func hasMTLS(p *peer.Peer) bool {
	if p == nil || p.AuthInfo == nil {
		return false
	}
	tlsInfo, ok := p.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return false
	}
	return len(tlsInfo.State.VerifiedChains) > 0 || len(tlsInfo.State.PeerCertificates) > 0
}
