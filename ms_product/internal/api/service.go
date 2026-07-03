package api

import (
	"ms_product/internal/core/config"
	"ms_product/internal/core/jsonlog"
	"ms_product/internal/core/security"
	"ms_product/internal/core/transaction"
)

type services struct {
	jwtService *security.JwtService
}

func NewServices(
	r *repositories,
	tx transaction.Manager,
	config config.Config,
	logger jsonlog.Logger,
) (*services, error) {
	// cacheClient, err := cache.NewRedisCache(config.Cache.Addr, config.Cache.Password, config.Cache.Db)

	// if err != nil {
	// 	return nil, err
	// }

	// logger.PrintInfo("reddis connection pool established", nil)

	jwtService, err := security.NewService(config)
	if err != nil {
		return nil, err
	}

	return &services{
		jwtService: jwtService,
	}, nil
}
