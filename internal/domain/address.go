package domain

// Address represents the domain entity for a postal address.
type Address struct {
	CEP    string `json:"cep"`
	Street string `json:"street"`
	City   string `json:"city"`
	State  string `json:"state"`
}
