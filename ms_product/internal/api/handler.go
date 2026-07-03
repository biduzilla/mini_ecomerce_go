package api

import (
	"ms_product/internal/core/domain/apiError"
	"ms_product/internal/features/product"
)

type handlers struct {
	*product.ProductHandler
}

func NewHandlers(
	services *services,
	errHandler *apiError.ErrorHandler,
) *handlers {
	return &handlers{
		ProductHandler: product.NewHandler(services.product, errHandler),
	}
}
