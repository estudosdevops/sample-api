---
applyTo: "**/*.go,go.mod,go.sum"
description: "Sample API observability guidelines"
---

# Sample API Instructions

This project is an observability playground.

Observability is more important than business logic.

Whenever implementing new code:

## Tracing

Always propagate context.Context.

Never create a new context.Background() inside request handlers.

Every request must belong to a trace.

Create manual spans for:

- Validation
- Business Logic
- PostgreSQL
- Redis
- External HTTP
- JSON Encoding

Keep spans small and meaningful.

Add attributes whenever possible.

Example:

- user.id
- cep
- cache.hit
- db.table
- db.operation
- external.service

Use span events for important events:

- cache hit
- cache miss
- retry
- timeout
- validation failed

Set span status correctly.

---

## Logs

Use slog.

Every log should contain:

- trace_id
- span_id
- request_id

Prefer structured logging.

Never use fmt.Println().

---

## Metrics

Whenever implementing a new feature think about metrics.

Prefer Counters for totals.

Prefer Histograms for latency.

Prefer Gauges for current values.

Metrics should help build RED dashboards.

---

## HTTP

Instrument every endpoint.

Return proper HTTP status codes.

Measure latency.

Record request size.

Record response size.

---

## PostgreSQL

Every query should execute inside a span.

Record:

- SQL operation
- table
- duration

Never log sensitive information.

---

## Redis

Instrument every command.

Record cache hit ratio.

Distinguish cache hit and cache miss.

---

## External APIs

Always propagate traceparent.

Measure request latency.

Record retries.

Record failures.

---

## Errors

Errors are learning opportunities.

Return useful errors.

Record them in traces.

Log them with context.

Increment error metrics.

---

## Learning

Whenever generating code:

Explain:

- why the span exists
- why the metric exists
- why the log exists

The explanation is as important as the implementation.