package viacep

import "context"

// Client defines the behaviour expected from an external ViaCEP client.
type Client interface {
	Lookup(ctx context.Context, cep string) (Response, error)
}

// Response is a minimal representation returned by ViaCEP.
type Response struct {
	CEP    string `json:"cep"`
	Street string `json:"logradouro"`
	City   string `json:"localidade"`
	State  string `json:"uf"`
}
