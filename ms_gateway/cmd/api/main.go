package main

import (
	"log"
	"ms_gateway/internal/api"
)

func main() {
	cfg := api.NewConfig()
	app := api.NewApp(cfg)

	if err := app.Serve(); err != nil {
		log.Fatal(err)
	}
}
