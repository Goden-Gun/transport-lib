package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

const traceMetadataKey = "x-trace-id"

var propagator = propagation.TraceContext{}

// InjectMetadata injects tracing context into gRPC metadata.
func InjectMetadata(ctx context.Context, md metadata.MD) metadata.MD {
	if md == nil {
		md = metadata.New(nil)
	}
	propagator.Inject(ctx, propagation.HeaderCarrier(md))
	if span := trace.SpanFromContext(ctx); span.SpanContext().HasTraceID() {
		md.Set(traceMetadataKey, span.SpanContext().TraceID().String())
	}
	return md
}

// ExtractMetadata extracts tracing context from metadata.
func ExtractMetadata(ctx context.Context, md metadata.MD) context.Context {
	if md == nil {
		return ctx
	}
	ctx = propagator.Extract(ctx, propagation.HeaderCarrier(md))
	if traceIDs := md.Get(traceMetadataKey); len(traceIDs) > 0 {
		span := trace.SpanFromContext(ctx)
		span.SetAttributes(attribute.String(traceMetadataKey, traceIDs[0]))
	}
	return ctx
}

// Tracer returns named tracer for transport components.
func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}
