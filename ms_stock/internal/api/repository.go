package api

import (
	"database/sql"

	"ms_stock/internal/core/jsonlog"
	"ms_stock/internal/features/stock"
)

type repositories struct {
	stockRepository *stock.StockRepository
}

func NewRepositories(
	db *sql.DB,
	logger jsonlog.Logger,
) *repositories {
	return &repositories{
		stockRepository: stock.NewRepository(db, logger),
	}
}
