package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/estudosdevops/sample-api/internal/domain"
	"github.com/estudosdevops/sample-api/internal/usecase"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	addrUC   *usecase.AddressUseCase
	postgres pinger
	redis    pinger
}

type pinger interface {
	Ping(context.Context) error
}

func NewHandler(auc *usecase.AddressUseCase, postgres, redis pinger) *Handler {
	return &Handler{addrUC: auc, postgres: postgres, redis: redis}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/address/:cep", h.getAddress)
	r.GET("/healthz", h.healthz)
	r.GET("/readyz", h.readyz)
}

func (h *Handler) healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

func (h *Handler) readyz(c *gin.Context) {
	type result struct {
		name   string
		status string
		err    error
	}
	results := make(chan result, 2)
	var checks sync.WaitGroup
	checks.Add(2)
	go func() {
		defer checks.Done()
		status, err := dependencyStatus(c.Request.Context(), h.postgres)
		results <- result{name: "postgres", status: status, err: err}
	}()
	go func() {
		defer checks.Done()
		status, err := dependencyStatus(c.Request.Context(), h.redis)
		results <- result{name: "redis", status: status, err: err}
	}()
	checks.Wait()
	close(results)

	postgresStatus, redisStatus := "down", "down"
	var postgresErr, redisErr error
	for check := range results {
		if check.name == "postgres" {
			postgresStatus = check.status
			postgresErr = check.err
		} else {
			redisStatus = check.status
			redisErr = check.err
		}
	}
	if postgresErr != nil || redisErr != nil {
		err := postgresErr
		if err == nil {
			err = redisErr
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "unhealthy",
			"postgres": postgresStatus,
			"redis":    redisStatus,
			"error":    err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "ok",
		"postgres": postgresStatus,
		"redis":    redisStatus,
	})
}

func dependencyStatus(ctx context.Context, dependency pinger) (string, error) {
	if dependency == nil {
		return "down", fmt.Errorf("dependency is not configured")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := dependency.Ping(pingCtx); err != nil {
		return "down", err
	}
	return "up", nil
}

func (h *Handler) getAddress(c *gin.Context) {
	cep := c.Param("cep")
	addr, err := h.addrUC.GetByCEP(c.Request.Context(), cep)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, addr)
}
