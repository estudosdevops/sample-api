package infra

import (
	"context"
	"os"

	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

var Logger *slog.Logger
var (
	ServiceName    string
	ServiceVersion string
	Environment    string
)

// InitLogger initializes a structured slog logger with service attributes.
func InitLogger(serviceName, serviceVersion, environment string) {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})
	ServiceName = serviceName
	ServiceVersion = serviceVersion
	Environment = environment
	Logger = slog.New(h).With(
		slog.String("service.name", serviceName),
		slog.String("service.version", serviceVersion),
		slog.String("environment", environment),
	)
}

// LoggerFromContext returns a logger enriched with trace/span/request identifiers from ctx.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if Logger == nil {
		// fallback to a basic logger
		h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{})
		Logger = slog.New(h)
	}
	sc := trace.SpanContextFromContext(ctx)
	l := Logger
	if sc.IsValid() {
		l = l.With(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	if reqID := RequestIDFromContext(ctx); reqID != "" {
		l = l.With(slog.String("request_id", reqID))
	}
	return l
}
