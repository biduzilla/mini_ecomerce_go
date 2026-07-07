package product

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ProductDTO struct {
	ID    *uuid.UUID `json:"id"`
	Name  *string    `json:"name"`
	Price *float64   `json:"price"`
}

type Client interface {
	GetByID(ctx context.Context, id uuid.UUID) (*ProductDTO, error)
}

type Config struct {
	BaseURL string
	Timeout time.Duration
}

var _ Client = (*HTTPClient)(nil)
