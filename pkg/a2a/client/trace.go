package client

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"google.golang.org/grpc/metadata"
)

func injectTraceContext(ctx context.Context) context.Context {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		md = metadata.MD{}
	} else {
		md = md.Copy()
	}
	otel.GetTextMapPropagator().Inject(ctx, metadataCarrier{md: md})
	return metadata.NewOutgoingContext(ctx, md)
}

type metadataCarrier struct {
	md metadata.MD
}

func (c metadataCarrier) Get(key string) string {
	values := c.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c metadataCarrier) Set(key, value string) {
	c.md.Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for key := range c.md {
		keys = append(keys, key)
	}
	return keys
}

var _ propagation.TextMapCarrier = metadataCarrier{}
