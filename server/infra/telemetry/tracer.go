package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitTracer(ctx context.Context) (func(), error) {
	// Writes spans to stdout — swap for OTLP/Jaeger later
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint("localhost:4318"),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	return func() {
		_ = tp.Shutdown(ctx)
	}, nil
}

// For production, swap stdout for OTLP:
//
// import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
//
// exporter, err := otlptracehttp.New(ctx,
//     otlptracehttp.WithEndpoint("localhost:4318"),
//     otlptracehttp.WithInsecure(),
// )
