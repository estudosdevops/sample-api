package infra

// otel bootstrap: OTLP exporter using gRPC for production-friendly Alloy ingestion.

import (
	"context"
	"net/http"
	"os"
	"strings"

	"log/slog"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	promexp "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitOpenTelemetry(ctx context.Context) (func(context.Context) error, error) {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		endpoint = "alloy:4317"
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			attribute.String("service.name", "sample-api"),
		),
	)
	if err != nil {
		return nil, err
	}

	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(bsp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	shutdown := func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}

	reg := prometheus.NewRegistry()
	promExp, err := promexp.New(promexp.WithRegisterer(reg))
	if err == nil {
		mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(promExp))
		otel.SetMeterProvider(mp)
		promRegistry = reg
	} else {
		slog.Warn("prometheus exporter init failed", "error", err)
	}

	return shutdown, nil
}

var promRegistry *prometheus.Registry

// MetricsHandler returns an http.Handler serving Prometheus metrics for the exporter registry.
func MetricsHandler() http.Handler {
	if promRegistry == nil {
		return promhttp.Handler()
	}
	return promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{})
}
