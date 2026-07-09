package stock

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AvailabilityCheckRequest struct {
	Items []ItemRequest `json:"items"`
}

type ItemRequest struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  int       `json:"quantity"`
}

type AvailabilityCheckResponse struct {
	Available bool                     `json:"available"`
	Details   []ItemAvailabilityDetail `json:"details"`
}

type ItemAvailabilityDetail struct {
	ProductID uuid.UUID `json:"productId"`
	Requested int       `json:"requested"`
	Available int       `json:"available"`
}

type Client interface {
	CheckAvailability(
		ctx context.Context,
		request AvailabilityCheckRequest,
	) (*AvailabilityCheckResponse, error)
}

type Config struct {
	BaseURL string
	Timeout time.Duration
}

var _ Client = (*HTTPClient)(nil)
