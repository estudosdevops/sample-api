package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/estudosdevops/sample-api/internal/clients/viacep"
	"github.com/estudosdevops/sample-api/internal/config"
	deliveryhttp "github.com/estudosdevops/sample-api/internal/delivery/http"
	"github.com/estudosdevops/sample-api/internal/infra"
	repo "github.com/estudosdevops/sample-api/internal/repository"
	memrepo "github.com/estudosdevops/sample-api/internal/repository/memory"
	postgresrepo "github.com/estudosdevops/sample-api/internal/repository/postgres"
	redisrepo "github.com/estudosdevops/sample-api/internal/repository/redis"
	"github.com/estudosdevops/sample-api/internal/usecase"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	infra.LoggerFromContext(context.Background()).Info("starting", "service.name", cfg.ServiceName, "service.version", cfg.ServiceVersion, "environment", cfg.Env)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// init logger
	infra.InitLogger(cfg.ServiceName, cfg.ServiceVersion, cfg.Env)

	// init OTEL with OTLP gRPC exporter to Alloy
	shutdownOtel, err := infra.InitOpenTelemetry(ctx)
	if err != nil {
		infra.LoggerFromContext(ctx).Warn("otel init failed", "error", err)
	}

	// create postgres client (scaffold)
	pg, err := postgresrepo.NewPostgres(cfg.PostgresDSN)
	if err != nil {
		infra.LoggerFromContext(ctx).Warn("postgres init failed", "error", err)
	} else {
		defer pg.Close()
	}

	// create redis client (scaffold)
	rr, err := redisrepo.NewRedis(cfg.RedisAddr)
	if err != nil {
		infra.LoggerFromContext(ctx).Warn("redis init failed", "error", err)
	} else {
		defer rr.Close()
	}

	// ping services with bounded retries and structured startup logs
	if pg != nil {
		infra.PingWithRetry(ctx, "postgres", cfg.PostgresDSN, pg.Ping)
	}
	if rr != nil {
		infra.PingWithRetry(ctx, "redis", cfg.RedisAddr, rr.Ping)
	}

	// choose address repository: prefer Postgres, fallback to memory
	var addrRepo repo.AddressRepository
	if pg != nil {
		addrRepo = pg
	} else {
		addrRepo = memrepo.NewMemoryRepo()
	}

	// cache repository (optional)
	var cacheRepo repo.CacheRepository
	if rr != nil {
		cacheRepo = rr
	}

	// external client for ViaCEP
	via := viacep.NewHTTPClient("https://viacep.com.br/ws")

	addrUC := usecase.NewAddressUseCase(addrRepo, cacheRepo, via)
	h := deliveryhttp.NewHandler(addrUC, pg, rr)

	// gin router
	r := gin.New()
	r.Use(gin.Recovery())
	// request middleware adds request_id and server spans
	r.Use(infra.RequestMiddleware(cfg.ServiceName, cfg.ServiceVersion, cfg.Env))
	h.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.HTTPPort),
		Handler: r,
	}

	// start server
	go func() {
		infra.LoggerFromContext(ctx).Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			infra.LoggerFromContext(ctx).Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// handle shutdown
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	infra.LoggerFromContext(ctx).Info("shutdown signal received")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		infra.LoggerFromContext(ctxShutdown).Warn("server shutdown error", "error", err)
	}
	if shutdownOtel != nil {
		if err := shutdownOtel(ctxShutdown); err != nil {
			infra.LoggerFromContext(ctxShutdown).Warn("otel shutdown error", "error", err)
		}
	}
	infra.LoggerFromContext(ctxShutdown).Info("server stopped")
}
