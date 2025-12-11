package bootstrap

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.26.0"

	"github.com/Goden-Gun/transport-lib/pkg/config"
)

// ShutdownFunc 关闭函数类型
type ShutdownFunc func(context.Context) error

// InitTracing 初始化 OpenTelemetry 分布式追踪
// 返回 shutdown 函数用于优雅关闭
func InitTracing(ctx context.Context, cfg config.TracingConfig) (ShutdownFunc, error) {
	exporterName := cfg.Exporter
	if exporterName == "" || exporterName == "disabled" {
		return func(context.Context) error { return nil }, nil
	}

	var (
		exporter sdktrace.SpanExporter
		err      error
	)

	switch exporterName {
	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp", "otlp-grpc":
		clientOpts := []otlptracegrpc.Option{}
		if cfg.Endpoint != "" {
			clientOpts = append(clientOpts, otlptracegrpc.WithEndpoint(cfg.Endpoint))
		}
		if cfg.Insecure {
			clientOpts = append(clientOpts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, clientOpts...)
	default:
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}
	if err != nil {
		return nil, err
	}

	serviceName := cfg.ServiceName
	if serviceName == "" {
		serviceName = "unknown-service"
	}

	attrs := []attribute.KeyValue{semconv.ServiceName(serviceName)}
	for k, v := range cfg.ResourceTags {
		attrs = append(attrs, attribute.String(k, v))
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(semconv.SchemaURL, attrs...))
	if err != nil {
		return nil, err
	}

	ratio := cfg.SampleRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 1
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}
