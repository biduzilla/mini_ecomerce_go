package api

import (
	"ms_stock/internal/core/cache"
	"ms_stock/internal/core/config"
	"ms_stock/internal/core/jsonlog"
	"ms_stock/internal/core/security"
	"ms_stock/internal/core/transaction"
	"ms_stock/internal/features/stock"
)

type services struct {
	jwtService   *security.JwtService
	stockService *stock.StockService
}

func NewServices(
	r *repositories,
	tx transaction.Manager,
	config config.Config,
	logger jsonlog.Logger,
	clients *clients,
) (*services, error) {
	cacheClient, err := cache.NewRedisCache(config.Cache.Addr, config.Cache.Password, config.Cache.Db)

	if err != nil {
		return nil, err
	}

	logger.PrintInfo("reddis connection pool established", nil)

	jwtService, err := security.NewService(config)
	if err != nil {
		return nil, err
	}

	return &services{
		jwtService: jwtService,
		stockService: stock.NewService(
			r.stockRepository,
			tx,
			cacheClient,
			cache.NewKeyBuilder("stocks"),
			clients.product,
		),
	}, nil
}
