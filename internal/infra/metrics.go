package infra

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// BusinessMetrics provides a clean API for recording business metrics.
// All metrics follow OpenTelemetry naming conventions.
type BusinessMetrics struct {
	cacheRequests metric.Int64Counter
	lookupsByUF   metric.Int64Counter
	dbOperations  metric.Int64Counter
}

var (
	metricsOnce     sync.Once
	businessMetrics *BusinessMetrics
)

// InitBusinessMetrics initializes business metrics once (singleton pattern).
func InitBusinessMetrics() *BusinessMetrics {
	metricsOnce.Do(func() {
		meter := otel.GetMeterProvider().Meter("sample-api")

		cacheReqs, err := meter.Int64Counter(
			"sample_api_cache_requests_total",
			metric.WithDescription("Total cache requests (hits and misses)"),
			metric.WithUnit("1"),
		)
		if err != nil {
			panic(err)
		}

		lookupsByUF, err := meter.Int64Counter(
			"sample_api_lookups_by_uf_total",
			metric.WithDescription("Total successful address lookups by state (UF)"),
			metric.WithUnit("1"),
		)
		if err != nil {
			panic(err)
		}

		dbOps, err := meter.Int64Counter(
			"sample_api_db_operations_total",
			metric.WithDescription("Total PostgreSQL operations (SELECT and INSERT)"),
			metric.WithUnit("1"),
		)
		if err != nil {
			panic(err)
		}

		businessMetrics = &BusinessMetrics{
			cacheRequests: cacheReqs,
			lookupsByUF:   lookupsByUF,
			dbOperations:  dbOps,
		}
	})
	return businessMetrics
}

// RecordCacheRequest records a cache hit or miss without boilerplate.
func (m *BusinessMetrics) RecordCacheRequest(ctx context.Context, isHit bool) {
	if m.cacheRequests == nil {
		return
	}
	result := "miss"
	if isHit {
		result = "hit"
	}
	m.cacheRequests.Add(ctx, 1, metric.WithAttributes(attribute.String("result", result)))
}

// RecordUFLookup records a successful address lookup by state and source.
func (m *BusinessMetrics) RecordUFLookup(ctx context.Context, uf, source string) {
	if m.lookupsByUF == nil {
		return
	}
	m.lookupsByUF.Add(ctx, 1, metric.WithAttributes(
		attribute.String("uf", uf),
		attribute.String("source", source),
	))
}

// RecordDBOp records a database operation with simplified error handling.
// Pass the error to automatically set status to "error" or "success".
func (m *BusinessMetrics) RecordDBOp(ctx context.Context, operation string, err error) {
	if m.dbOperations == nil {
		return
	}
	status := "success"
	if err != nil {
		status = "error"
	}
	m.dbOperations.Add(ctx, 1, metric.WithAttributes(
		attribute.String("operation", operation),
		attribute.String("status", status),
	))
}

// GetBusinessMetrics returns the singleton instance of business metrics.
func GetBusinessMetrics() *BusinessMetrics {
	return businessMetrics
}
