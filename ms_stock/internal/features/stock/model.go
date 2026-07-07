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
	AvailableQuantity int32
}

type StockDTO struct {
	ID                *uuid.UUID `json:"id"`
	ProductId         *uuid.UUID `json:"product_id"`
	AvailableQuantity *int32     `json:"available_quantity"`
	Version           *int       `json:"version"`
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
