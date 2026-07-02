package api

import (
	"ms_auth/internal/core/cache"
	"ms_auth/internal/core/config"
	"ms_auth/internal/core/jsonlog"
	"ms_auth/internal/core/security"
	"ms_auth/internal/core/transaction"
	"ms_auth/internal/features/auth"
	"ms_auth/internal/features/user"
)

type services struct {
	jwtService  *security.JwtService
	userService *user.UserService
	authService *auth.AuthService
}

func NewServices(
	r *repositories,
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
	userService := user.NewService(&r.userRepository, tx, cacheClient, cache.NewKeyBuilder("user"))

	return &services{
		jwtService:  jwtService,
		userService: userService,
		authService: auth.NewService(userService, jwtService),
	}, nil
}
