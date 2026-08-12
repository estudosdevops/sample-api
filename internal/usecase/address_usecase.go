package usecase

import (
	"context"
	"errors"
	"strings"
	"unicode"

	"github.com/estudosdevops/sample-api/internal/clients/viacep"
	"github.com/estudosdevops/sample-api/internal/domain"
	"github.com/estudosdevops/sample-api/internal/infra"
	"github.com/estudosdevops/sample-api/internal/repository"
)

// AddressUseCase implements business logic for addresses.
type AddressUseCase struct {
	repo    repository.AddressRepository
	cache   repository.CacheRepository
	viacep  viacep.Client
	metrics *infra.BusinessMetrics
}

func NewAddressUseCase(r repository.AddressRepository, c repository.CacheRepository, v viacep.Client) *AddressUseCase {
	return &AddressUseCase{
		repo:    r,
		cache:   c,
		viacep:  v,
		metrics: infra.InitBusinessMetrics(),
	}
}

func (uc *AddressUseCase) GetByCEP(ctx context.Context, cep string) (*domain.Address, error) {
	cep = sanitizeCEP(cep)
	if cep == "" {
		return nil, domain.ErrValidation
	}

	// check cache first
	if uc.cache != nil {
		if a, err := uc.cache.GetAddress(ctx, cep); err == nil {
			uc.metrics.RecordCacheRequest(ctx, true) // hit
			uc.metrics.RecordUFLookup(ctx, a.State, "redis")
			return a, nil
		}
		uc.metrics.RecordCacheRequest(ctx, false) // miss
	}

	// fallback to repository (DB)
	addr, err := uc.repo.GetByCEP(ctx, cep)
	if err == nil {
		uc.metrics.RecordUFLookup(ctx, addr.State, "postgres")
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
		CEP:    sanitizeCEP(resp.CEP),
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

	uc.metrics.RecordUFLookup(ctx, newAddr.State, "viacep")
	return newAddr, nil
}

func sanitizeCEP(cep string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, cep)
}
