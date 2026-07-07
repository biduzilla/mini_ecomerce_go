package main

import (
	"ms_order/internal/api"
	"ms_order/internal/core/config"
)

func main() {
	cfg := config.New()
	cfg.Env = "development"

	app := api.NewApp(*cfg)
	err := app.Server()
	if err != nil {
		app.Logger.PrintFatal(err, nil)
	}
}
