package infra

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

// RequestMiddleware returns a Gin middleware that ensures a request_id is present,
// starts a server span for the request and enriches it with standard attributes.
func RequestMiddleware(serviceName, serviceVersion, environment string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// obtain or generate request id
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = c.GetHeader("X-Request-ID")
		}
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// start server span
		tracer := otel.Tracer("http-server")
		ctx, span := tracer.Start(c.Request.Context(), fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()))
		span.SetAttributes(
			attribute.String("request_id", reqID),
			attribute.String("http.method", c.Request.Method),
			attribute.String("http.target", c.Request.URL.Path),
			attribute.String("service.name", serviceName),
			attribute.String("service.version", serviceVersion),
			attribute.String("environment", environment),
		)

		// attach request id to context
		ctx = contextWithRequestID(ctx, reqID)
		c.Request = c.Request.WithContext(ctx)

		// ensure response header includes request id
		c.Writer.Header().Set("X-Request-Id", reqID)

		// proceed and finish span after handlers
		c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
		span.End()
	}
}
