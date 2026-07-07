package main

import (
	"ms_stock/internal/api"
	"ms_stock/internal/core/config"
	"time"
)

func main() {
	// c := config.New()
	var cfg config.Config

	// cfg.Server.Port = c.Server.Port
	// cfg.Server.Timeout = c.Server.Timeout
	// cfg.Env = "development"
	// cfg.DB.DSN = c.DB.DSN
	// cfg.DB.MaxOpenConns = c.DB.MaxOpenConns
	// cfg.DB.MaxIdleConns = c.DB.MaxIdleConns
	// cfg.DB.MaxIdleTime = c.DB.MaxIdleTime
	// cfg.Limiter.RPS = c.RateLimiter.RPS
	// cfg.Limiter.Burst = c.RateLimiter.Burst
	// cfg.Limiter.Enabled = c.RateLimiter.Enabled
	// cfg.Security.PrivateKeyPath = c.Security.PrivateKeyPath
	// cfg.Security.PublicKeyPath = c.Security.PublicKeyPath
	// cfg.Cache.Addr = c.Cache.Addr
	// cfg.Cache.Password = c.Cache.Password
	// cfg.Cache.Db = c.Cache.Db
	// cfg.Clients.ProductURL = c.Clients.ProductURL

	cfg.Server.Port = 4003
	cfg.Server.Timeout = 5 * time.Second
	cfg.Env = "development"
	cfg.DB.DSN = "postgres://api_user:api_password@postgres:5432/api_db?sslmode=disable"
	cfg.DB.MaxOpenConns = 25
	cfg.DB.MaxIdleConns = 25
	cfg.DB.MaxIdleTime = "15m"
	cfg.Limiter.RPS = 2.0
	cfg.Limiter.Burst = 4
	cfg.Limiter.Enabled = true
	cfg.Security.PrivateKeyPath = "resources/keys/privateKey.pem"
	cfg.Security.PublicKeyPath = "resources/keys/publicKey.pem"
	cfg.Cache.Addr = "redis:6379"
	cfg.Cache.Password = "redis_secure_password"
	cfg.Cache.Db = 0
	cfg.Clients.ProductURL = "http://localhost:4002/v1/product/"

	app := api.NewApp(cfg)
	err := app.Server()
	if err != nil {
		app.Logger.PrintFatal(err, nil)
	}
}
