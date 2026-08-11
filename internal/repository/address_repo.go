package repository

import (
	"context"

	"github.com/estudosdevops/sample-api/internal/domain"
)

// AddressRepository defines persistence operations for Address.
type AddressRepository interface {
	GetByCEP(ctx context.Context, cep string) (*domain.Address, error)
	Insert(ctx context.Context, a *domain.Address) error
}

// TxController provides transaction control when needed by usecases.
type TxController interface {
	BeginTx(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
