package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type healthPinger struct {
	err error
}

func (p healthPinger) Ping(context.Context) error {
	return p.err
}

func TestHealthz(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, healthPinger{}, healthPinger{}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "healthy" {
		t.Fatalf("expected healthy status, got %q", body["status"])
	}
}

func TestReadyzUnavailableDependency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, healthPinger{err: errors.New("postgres unavailable")}, healthPinger{}).RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", res.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "unhealthy" || body["postgres"] != "down" || body["redis"] != "up" {
		t.Fatalf("unexpected readiness response: %#v", body)
	}
	if body["error"] == "" {
		t.Fatal("expected readiness error")
	}
}
