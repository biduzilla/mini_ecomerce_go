package main

import (
	"ms_stock/internal/api"
	"ms_stock/internal/core/config"
	"ms_stock/internal/core/jsonlog"
	"os"
)

func main() {
	cfg := config.New()
	cfg.Env = "development"
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
