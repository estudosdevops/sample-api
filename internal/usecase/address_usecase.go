package usecase

import (
	"context"
	"errors"
	"sync"

	"github.com/estudosdevops/sample-api/internal/clients/viacep"
	"github.com/estudosdevops/sample-api/internal/domain"
	"github.com/estudosdevops/sample-api/internal/infra"
	"github.com/estudosdevops/sample-api/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	metric "go.opentelemetry.io/otel/metric"
)

// AddressUseCase implements business logic for addresses.
type AddressUseCase struct {
	repo   repository.AddressRepository
	cache  repository.CacheRepository
	viacep viacep.Client
}

var (
	metricsOnce     sync.Once
	cacheHitCounter interface {
		Add(context.Context, int64, ...metric.AddOption)
	}
	cacheMissCounter interface {
		Add(context.Context, int64, ...metric.AddOption)
	}
)

func initMetrics() {
	metricsOnce.Do(func() {
		meter := otel.GetMeterProvider().Meter("sample-api")
		// metric names follow prometheus conventions: <namespace>_<subsystem>_<name>_total
		if c, err := meter.Int64Counter("sample_api_redis_cache_hits_total"); err == nil {
			cacheHitCounter = c
		}
		if c, err := meter.Int64Counter("sample_api_redis_cache_misses_total"); err == nil {
			cacheMissCounter = c
		}
	})
}

func NewAddressUseCase(r repository.AddressRepository, c repository.CacheRepository, v viacep.Client) *AddressUseCase {
	initMetrics()
	return &AddressUseCase{repo: r, cache: c, viacep: v}
}

func (uc *AddressUseCase) GetByCEP(ctx context.Context, cep string) (*domain.Address, error) {
	if cep == "" {
		return nil, domain.ErrValidation
	}

	// check cache first
	if uc.cache != nil {
		if a, err := uc.cache.GetAddress(ctx, cep); err == nil {
			if cacheHitCounter != nil {
				cacheHitCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("service.version", infra.ServiceVersion), attribute.String("environment", infra.Environment)))
			}
			return a, nil
		}
		if cacheMissCounter != nil {
			cacheMissCounter.Add(ctx, 1, metric.WithAttributes(attribute.String("service.version", infra.ServiceVersion), attribute.String("environment", infra.Environment)))
		}
	}

	// fallback to repository (DB)
	addr, err := uc.repo.GetByCEP(ctx, cep)
	if err == nil {
		// populate cache
		if uc.cache != nil {
			_ = uc.cache.SetAddress(ctx, addr)
		}
		return addr, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	// not found in DB -> ask external ViaCEP
	if uc.viacep == nil {
		return nil, domain.ErrNotFound
	}
	resp, err := uc.viacep.Lookup(ctx, cep)
	if err != nil {
		return nil, err
	}
	newAddr := &domain.Address{
		CEP:    resp.CEP,
		Street: resp.Street,
		City:   resp.City,
		State:  resp.State,
	}

	// persist to DB (best-effort)
	if perr := uc.repo.Insert(ctx, newAddr); perr != nil {
		infra.LoggerFromContext(ctx).Warn("failed to persist address", "error", perr, "cep", cep)
	}

	// populate cache (best-effort)
	if uc.cache != nil {
		if cerr := uc.cache.SetAddress(ctx, newAddr); cerr != nil {
			infra.LoggerFromContext(ctx).Warn("failed to set cache", "error", cerr, "cep", cep)
		}
	}

	return newAddr, nil
}
