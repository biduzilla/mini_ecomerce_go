package main

import (
	"ms_stock/internal/api"
	"ms_stock/internal/core/config"
	"ms_stock/internal/core/jsonlog"
	"os"
)

func main() {
	// var cfg config.Config

	cfg := config.New()
	cfg.Env = "development"

	// cfg.Server.Port = 4003
	// cfg.Server.Timeout = 5 * time.Second
	// cfg.Env = "development"
	// cfg.DB.DSN = "postgres://api_user:api_password@localhost:5432/api_db?sslmode=disable"
	// cfg.DB.MaxOpenConns = 25
	// cfg.DB.MaxIdleConns = 25
	// cfg.DB.MaxIdleTime = "15m"
	// cfg.Limiter.RPS = 2.0
	// cfg.Limiter.Burst = 4
	// cfg.Limiter.Enabled = true
	// cfg.Security.PrivateKeyPath = "resources/keys/privateKey.pem"
	// cfg.Security.PublicKeyPath = "resources/keys/publicKey.pem"
	// cfg.Cache.Addr = "localhost:6379"
	// cfg.Cache.Password = "redis_secure_password"
	// cfg.Cache.Db = 0
	// cfg.Clients.ProductURL = "http://localhost:4002/v1/product"
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	app, err := api.NewApp(*cfg, logger)
	if app == nil {
		logger.PrintError(err, map[string]string{
			"message": "failed to initialize app",
		})
	}
	err = app.Server()
	if err != nil {
		app.Logger.PrintFatal(err, nil)
	}
}
