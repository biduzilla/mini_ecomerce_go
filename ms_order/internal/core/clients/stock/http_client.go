package stock

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"ms_order/internal/core/contexts"
	"ms_order/internal/core/domain/apiError"
	"net/http"
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

func (c *HTTPClient) CheckAvailability(
	ctx context.Context,
	request AvailabilityCheckRequest,
) (*AvailabilityCheckResponse, error) {
	url := fmt.Sprintf("%s", c.baseURL)
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal availability check request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	token := contexts.GetToken(ctx)
	if token == "" {
		return nil, fmt.Errorf("internal error: attempted to call stock service without an authentication token in context")
	}

	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return nil, apiError.NewApiError(
			"insufficient stock for requested items",
			http.StatusUnprocessableEntity,
		)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("stock service returned unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var response AvailabilityCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode stock availability response: %w", err)
	}

	return &response, nil
}
