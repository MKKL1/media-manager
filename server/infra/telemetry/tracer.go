package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.40.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const tracerName = "server"

type Config struct {
	Enabled     bool
	Endpoint    string
	ServiceName string
}

func Init(ctx context.Context, cfg Config) (shutdown func(), err error) {
	if !cfg.Enabled {
		return func() {}, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return func() { _ = tp.Shutdown(ctx) }, nil
}

func Start(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, func(*error)) {
	ctx, span := otel.Tracer(tracerName).Start(ctx, name, opts...)
	return ctx, func(errp *error) {
		if errp != nil && *errp != nil {
			span.RecordError(*errp)
			span.SetStatus(codes.Error, (*errp).Error())
		}
		span.End()
	}
}
