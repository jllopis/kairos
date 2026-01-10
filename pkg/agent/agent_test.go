package agent

import (
	"context"
	"testing"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/memory"
)

func TestAgentMemoryInjection(t *testing.T) {
	mem := memory.NewInMemory()
	a, err := New(
		"agent-1",
		WithMemory(mem),
		WithHandler(func(ctx context.Context, input any) (any, error) {
			got, ok := core.MemoryFromContext(ctx)
			if !ok {
				t.Fatal("expected memory in context")
			}
			if got != mem {
				t.Fatal("memory instance mismatch")
			}
			return "ok", nil
		}),
	)
	if err != nil {
		t.Fatalf("agent creation failed: %v", err)
	}

	if _, err := a.Run(context.Background(), nil); err != nil {
		t.Fatalf("agent run failed: %v", err)
	}
}
