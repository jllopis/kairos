package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"testing"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func TestAuthenticator_RequiresOIDC(t *testing.T) {
	auth := NewAuthenticator(AuthConfig{RequireOIDC: true})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})
	if err := auth.Authenticate(ctx, authContext(ctx, "/a2a.v1.A2AService/SendMessage")); err == nil {
		t.Fatalf("expected missing bearer token error")
	}
}

func TestAuthenticator_OIDCWithBearer(t *testing.T) {
	auth := NewAuthenticator(AuthConfig{RequireOIDC: true})
	md := metadata.MD{"authorization": []string{"Bearer token-123"}}
	ctx := metadata.NewIncomingContext(context.Background(), md)
	if err := auth.Authenticate(ctx, authContext(ctx, "/a2a.v1.A2AService/SendMessage")); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func TestAuthenticator_RequiresMTLS(t *testing.T) {
	auth := NewAuthenticator(AuthConfig{RequireMTLS: true})
	ctx := context.Background()
	if err := auth.Authenticate(ctx, authContext(ctx, "/a2a.v1.A2AService/SendMessage")); err == nil {
		t.Fatalf("expected mutual TLS required error")
	}
}

func TestAuthenticator_MTLSWithPeer(t *testing.T) {
	auth := NewAuthenticator(AuthConfig{RequireMTLS: true})
	p := peer.Peer{AuthInfo: credentials.TLSInfo{State: tlsInfoWithPeerCert()}}
	ctx := peer.NewContext(context.Background(), &p)
	if err := auth.Authenticate(ctx, authContext(ctx, "/a2a.v1.A2AService/SendMessage")); err != nil {
		t.Fatalf("expected ok, got %v", err)
	}
}

func tlsInfoWithPeerCert() tls.ConnectionState {
	return tls.ConnectionState{
		PeerCertificates: []*x509.Certificate{{}},
	}
}
