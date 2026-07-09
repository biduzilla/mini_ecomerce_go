package api

import (
	"ms_order/internal/core/clients/stock"
	"ms_order/internal/core/config"
)

type clients struct {
	stockClient stock.Client
}

func NewClients(
	cfg config.Config,
) *clients {
	return &clients{
		stockClient: stock.NewClient(
			stock.Config{
				BaseURL: cfg.Clients.StockURL,
				Timeout: cfg.Server.Timeout,
			},
		),
	}
}
