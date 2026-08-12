package infra

import (
	"context"
	"strings"
	"time"
)

const (
	dependencyAttempts = 3
	dependencyTimeout  = 2 * time.Second
	dependencyInterval = 2 * time.Second
)

// PingWithRetry checks a dependency during startup and logs each attempt.
func PingWithRetry(ctx context.Context, name, address string, ping func(context.Context) error) bool {
	for attempt := 1; attempt <= dependencyAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, dependencyTimeout)
		startedAt := time.Now()
		err := ping(pingCtx)
		latency := time.Since(startedAt)
		cancel()

		if err == nil {
			LoggerFromContext(ctx).Info("dependency connected",
				"dependency", name,
				"address", redactAddress(address),
				"attempt", attempt,
				"latency_ms", latency.Milliseconds(),
				"status", "up",
			)
			return true
		}

		LoggerFromContext(ctx).Warn("dependency connection failed",
			"dependency", name,
			"address", redactAddress(address),
			"attempt", attempt,
			"latency_ms", latency.Milliseconds(),
			"status", "down",
			"error", err,
		)
		if attempt < dependencyAttempts {
			select {
			case <-time.After(dependencyInterval):
			case <-ctx.Done():
				return false
			}
		}
	}
	return false
}

func redactAddress(address string) string {
	if at := strings.LastIndex(address, "@"); at >= 0 {
		return address[at+1:]
	}
	return address
}
