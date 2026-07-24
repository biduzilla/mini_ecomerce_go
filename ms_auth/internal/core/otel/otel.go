package otel

import (
	"context"
	"fmt"
	"ms_auth/internal/core/jsonlog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func InitTracer(serviceName string, logger jsonlog.Logger) (func(context.Context) error, error) {
	exp, err := otlptracehttp.New(context.Background())
	if err != nil {
		return nil, fmt.Errorf("criando exportador OTLP HTTP: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.PrintError(err, nil)
	}))

	return tp.Shutdown, nil
}
