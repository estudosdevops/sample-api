package repository

import (
	"context"

	"github.com/estudosdevops/sample-api/internal/domain"
)

// CacheRepository defines cache operations for Address.
type CacheRepository interface {
	GetAddress(ctx context.Context, cep string) (*domain.Address, error)
	SetAddress(ctx context.Context, a *domain.Address) error
}
