package infra

// otel bootstrap: OTLP exporter using gRPC for production-friendly Alloy ingestion.

import (
	"context"
	"os"
	"strings"
	"time"

	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func InitOpenTelemetry(ctx context.Context) (func(context.Context) error, error) {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		endpoint = "alloy:4317"
	}

	// use a short timeout for exporter initialization to avoid blocking startup
	initCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	traceExporter, err := otlptracegrpc.New(initCtx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		slog.Warn("trace exporter init failed (using no-op)", "error", err)
		// use no-op exporter to avoid panics, startup continues
		traceExporter = nil
	}

	res, err := sdkresource.New(ctx,
		sdkresource.WithAttributes(
			attribute.String("service.name", "sample-api"),
		),
	)
	if err != nil {
		return nil, err
	}

	var tp *sdktrace.TracerProvider
	if traceExporter != nil {
		bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
		tp = sdktrace.NewTracerProvider(
			sdktrace.WithSpanProcessor(bsp),
			sdktrace.WithResource(res),
		)
	} else {
		// no-op tracer provider if exporter failed
		tp = sdktrace.NewTracerProvider(sdktrace.WithResource(res))
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// metric exporter via OTLP gRPC to Alloy
	metricExporter, err := otlpmetricgrpc.New(initCtx,
		otlpmetricgrpc.WithEndpoint(endpoint),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		slog.Warn("metric exporter init failed (using no-op)", "error", err)
		metricExporter = nil
	}

	// meter provider with periodic reader
	var mp *metric.MeterProvider
	if metricExporter != nil {
		reader := metric.NewPeriodicReader(metricExporter)
		mp = metric.NewMeterProvider(
			metric.WithReader(reader),
			metric.WithResource(res),
		)
	} else {
		// no-op meter provider if exporter failed
		mp = metric.NewMeterProvider(metric.WithResource(res))
	}
	otel.SetMeterProvider(mp)

	shutdown := func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Warn("tracer provider shutdown failed", "error", err)
		}
		if err := mp.Shutdown(ctx); err != nil {
			slog.Warn("meter provider shutdown failed", "error", err)
		}
		return nil
	}

	return shutdown, nil
}
