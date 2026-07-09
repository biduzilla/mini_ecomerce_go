package api

import (
	"ms_order/internal/core/cache"
	"ms_order/internal/core/config"
	"ms_order/internal/core/jsonlog"
	"ms_order/internal/core/security"
	"ms_order/internal/core/transaction"
	"ms_order/internal/features/order"
)

type services struct {
	jwtService   *security.JwtService
	orderService *order.OrderService
}

func NewServices(
	r *repositories,
	clients *clients,
	procecers *producers,
	tx transaction.Manager,
	config config.Config,
	logger jsonlog.Logger,
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

	orderService := order.NewService(r.OrderRepository,
		tx, cacheClient,
		cache.NewKeyBuilder("orders"),
		clients.stockClient,
		procecers.orderProducer,
		logger,
	)

	return &services{
		jwtService:   jwtService,
		orderService: orderService,
	}, nil
}
