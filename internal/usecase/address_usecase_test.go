package usecase

import (
	"context"
	"testing"

	"github.com/estudosdevops/sample-api/internal/domain"
)

type stubRepo struct {
	data map[string]*domain.Address
}

func (s *stubRepo) GetByCEP(ctx context.Context, cep string) (*domain.Address, error) {
	if a, ok := s.data[cep]; ok {
		return a, nil
	}
	return nil, domain.ErrNotFound
}

func (s *stubRepo) Insert(ctx context.Context, a *domain.Address) error {
	s.data[a.CEP] = a
	return nil
}

func TestGetByCEP_Validation(t *testing.T) {
	uc := NewAddressUseCase(&stubRepo{data: map[string]*domain.Address{}}, nil, nil)
	_, err := uc.GetByCEP(context.Background(), "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err != domain.ErrValidation {
		t.Fatalf("expected ErrValidation, got %v", err)
	}
}

func TestGetByCEP_Found(t *testing.T) {
	repo := &stubRepo{data: map[string]*domain.Address{"01001000": {CEP: "01001000", Street: "Praça da Sé", City: "São Paulo", State: "SP"}}}
	uc := NewAddressUseCase(repo, nil, nil)
	a, err := uc.GetByCEP(context.Background(), "01001000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.CEP != "01001000" {
		t.Fatalf("unexpected cep: %s", a.CEP)
	}
}
