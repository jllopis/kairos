package main

import (
	"context"
	"fmt"

	"github.com/jllopis/kairos/pkg/agent"
	"github.com/jllopis/kairos/pkg/memory"
	"github.com/jllopis/kairos/pkg/runtime"
)

func main() {
	mem := memory.NewFileStore("./.kairos/memory.jsonl")
	helloAgent, err := agent.New(
		"hello-agent",
		agent.WithRole("Greeter"),
		agent.WithMemory(mem),
		agent.WithHandler(func(ctx context.Context, input any) (any, error) {
			name, _ := input.(string)
			if name == "" {
				name = "world"
			}
			if err := mem.Store(ctx, map[string]any{"name": name}); err != nil {
				return nil, err
			}
			return fmt.Sprintf("hello, %s", name), nil
		}),
	)
	if err != nil {
		panic(err)
	}

	rt := runtime.NewLocal()
	if err := rt.Start(context.Background()); err != nil {
		panic(err)
	}
	defer func() {
		_ = rt.Stop(context.Background())
	}()

	output, err := rt.Run(context.Background(), helloAgent, "kairos")
	if err != nil {
		panic(err)
	}
	fmt.Println(output)
}
