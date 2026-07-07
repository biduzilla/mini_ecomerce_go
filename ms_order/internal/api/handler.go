package api

import (
	"ms_order/internal/core/domain/apiError"
)

type handlers struct {
}

func NewHandlers(
	services *services,
	errHandler *apiError.ErrorHandler,
) *handlers {
	return &handlers{}
}
