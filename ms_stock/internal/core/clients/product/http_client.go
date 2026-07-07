package product

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"ms_stock/internal/core/contexts"
	"net/http"

	"github.com/google/uuid"
)

type HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(cfg Config) *HTTPClient {
	return &HTTPClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

func (c *HTTPClient) GetByID(ctx context.Context, id uuid.UUID) (*ProductDTO, error) {
	url := fmt.Sprintf("%s/%s", c.baseURL, id.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	token := contexts.GetToken(ctx)
	if token == "" {
		return nil, fmt.Errorf("internal error: attempted to cal product service without an authentication token in context")
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrProcutNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("product service returned unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var dto ProductDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf("failed to decode product response: %w", err)
	}

	return &dto, nil
}

var ErrProcutNotFound = fmt.Errorf("product not found in product service")
