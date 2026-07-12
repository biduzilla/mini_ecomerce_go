package main

import (
	"ms_product/internal/api"
	"ms_product/internal/core/config"
)

func main() {
	c := config.New()
	c.Env = "development"
	app := api.NewApp(*c)
	err := app.Server()
	if err != nil {
		app.Logger.PrintFatal(err, nil)
	}
}
