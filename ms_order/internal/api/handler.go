package api

import (
	"ms_order/internal/core/domain/apiError"
	"ms_order/internal/features/order"
)

type handlers struct {
	*order.OrderHandler
}

func NewHandlers(
	services *services,
	errHandler *apiError.ErrorHandler,
) *handlers {
	return &handlers{
		OrderHandler: order.NewHandler(services.orderService, errHandler),
	}
}
