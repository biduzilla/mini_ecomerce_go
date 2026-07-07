package stock

import (
	"ms_stock/internal/core/domain/models"
	"ms_stock/internal/core/validator"

	"github.com/google/uuid"
)

type Stock struct {
	models.BaseModel
	ID                uuid.UUID
	ProductId         uuid.UUID
	AvailableQuantity int
}

type StockDTO struct {
	ID                *uuid.UUID `json:"id"`
	ProductId         *uuid.UUID `json:"product_id"`
	AvailableQuantity *int       `json:"available_quantity"`
	Version           *int       `json:"version"`
}

type ItemRequest struct {
	ProductID uuid.UUID `json:"productId"`
	Quantity  int       `json:"quantity"`
}

type AvailabilityCheckRequest struct {
	Items []ItemRequest `json:"items"`
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

func (m *Stock) ToDTO() *StockDTO {
	return &StockDTO{
		ID:                &m.ID,
		ProductId:         &m.ProductId,
		AvailableQuantity: &m.AvailableQuantity,
		Version:           &m.Version,
	}
}

func (d *StockDTO) ToModel() *Stock {
	var model Stock

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.ProductId != nil {
		model.ProductId = *d.ProductId
	}

	if d.AvailableQuantity != nil {
		model.AvailableQuantity = *d.AvailableQuantity
	}

	if d.Version != nil {
		model.Version = *d.Version
	}

	return &model
}

func (u *Stock) Validate(v *validator.Validator) {
	v.Check(u.ProductId != uuid.Nil, "product_id", "must be provided")
	v.Check(u.AvailableQuantity >= 0, "available_quantity", "must be greater than or equal to 0")
}

func (r *AvailabilityCheckRequest) Validate(v *validator.Validator) {
	v.Check(len(r.Items) > 0, "items", "must contain at least one item")

	for _, item := range r.Items {
		item.Validate(v)
	}
}

func (i *ItemRequest) Validate(v *validator.Validator) {
	v.Check(i.ProductID != uuid.Nil, "productId", "is required")
	v.Check(i.Quantity > 0, "quantity", "must be at least 1")
}
