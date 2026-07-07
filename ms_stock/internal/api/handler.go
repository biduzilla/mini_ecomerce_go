package api

import (
	"ms_stock/internal/core/domain/apiError"
	"ms_stock/internal/features/stock"
)

type handlers struct {
	stockHandler *stock.StockHandler
}

func NewHandlers(
	services *services,
	errHandler *apiError.ErrorHandler,
) *handlers {
	return &handlers{
		stockHandler: stock.NewHandler(services.stockService, errHandler),
	}
}
