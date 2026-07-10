package api

import (
	"ms_stock/internal/core/clients/product"
	"ms_stock/internal/core/config"
)

type clients struct {
	product *product.HTTPClient
}

func NewClients(
	config config.Config,
) *clients {
	return &clients{
		product: product.NewClient(product.Config{
			BaseURL: config.Clients.ProductURL,
			Timeout: config.Server.Timeout,
		}),
	}
}
