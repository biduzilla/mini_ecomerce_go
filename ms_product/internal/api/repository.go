package api

import (
	"database/sql"
	"ms_product/internal/core/jsonlog"
	"ms_product/internal/features/product"
)

type repositories struct {
	product *product.ProductRepository
}

func NewRepositories(
	db *sql.DB,
	logger jsonlog.Logger,
) *repositories {
	return &repositories{
		product: product.NewRepository(db, logger),
	}
}
