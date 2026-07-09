package api

import (
	"database/sql"

	"ms_order/internal/core/jsonlog"
	"ms_order/internal/features/order"
)

type repositories struct {
	*order.OrderRepository
}

func NewRepositories(
	db *sql.DB,
	logger jsonlog.Logger,
) *repositories {
	return &repositories{
		OrderRepository: order.NewRepository(db, logger),
	}
}
