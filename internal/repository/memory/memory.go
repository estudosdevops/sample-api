package memory

import (
	"context"
	"sync"

	"github.com/estudosdevops/sample-api/internal/domain"
	"github.com/estudosdevops/sample-api/internal/repository"
)

type MemoryRepo struct {
	mu    sync.RWMutex
	store map[string]*domain.Address
}

func NewMemoryRepo() repository.AddressRepository {
	return &MemoryRepo{store: make(map[string]*domain.Address)}
}

func (m *MemoryRepo) GetByCEP(ctx context.Context, cep string) (*domain.Address, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if a, ok := m.store[cep]; ok {
		return a, nil
	}
	return nil, domain.ErrNotFound
}

func (m *MemoryRepo) Insert(ctx context.Context, a *domain.Address) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[a.CEP] = a
	return nil
}
