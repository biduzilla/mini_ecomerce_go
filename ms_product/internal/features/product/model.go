package product

import (
	"ms_product/internal/core/domain/models"
	"ms_product/internal/core/validator"

	"github.com/google/uuid"
)

type Product struct {
	models.BaseModel
	ID    uuid.UUID
	Name  string
	Price float64
}

type ProductDTO struct {
	ID          *uuid.UUID `json:"id"`
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	Price       *float64   `json:"price"`
	Version     *int       `json:"version"`
}

func (m *Product) ToDTO() *ProductDTO {
	return &ProductDTO{
		ID:      &m.ID,
		Name:    &m.Name,
		Price:   &m.Price,
		Version: &m.Version,
	}
}

func (d ProductDTO) ToModel() (*Product, error) {
	var model Product

	if d.ID != nil {
		model.ID = *d.ID
	}
	if d.Name != nil {
		model.Name = *d.Name
	}
	if d.Price != nil {
		model.Price = *d.Price
	}
	if d.Version != nil {
		model.Version = *d.Version
	}

	return &model, nil
}

func (u *Product) Validate(v *validator.Validator) {
	v.Check(u.Name != "", "name", "must be provided")
	v.Check(len(u.Name) >= 3, "name", "must be at least 3 characters long")
	v.Check(len(u.Name) <= 200, "name", "must not be more than 200 characters long")
	v.Check(u.Price > 0, "price", "must be greater than 0")
}
