package viacep

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type httpClient struct {
	endpoint string
	client   *http.Client
}

// NewHTTPClient creates a ViaCEP client. endpoint should be like "https://viacep.com.br/ws"
func NewHTTPClient(endpoint string) Client {
	return &httpClient{
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (h *httpClient) Lookup(ctx context.Context, cep string) (Response, error) {
	var resp Response
	url := fmt.Sprintf("%s/%s/json/", h.endpoint, cep)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return resp, err
	}
	r, err := h.client.Do(req)
	if err != nil {
		return resp, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("viacep status: %d", r.StatusCode)
	}
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		return resp, err
	}
	return resp, nil
}
