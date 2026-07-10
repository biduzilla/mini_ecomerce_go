package api

import (
	"fmt"
	"ms_gateway/internal/core/jsonlog"
	"net/http"
	"os"
)

type application struct {
	config *Config
	logger jsonlog.Logger
}

func NewApp(cfg *Config) *application {
	return &application{
		config: cfg,
	}
}

func (app *application) Serve() error {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	app.logger = logger

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.config.Server.Port),
		Handler: app.Routes(),
	}

	app.logger.PrintInfo(fmt.Sprintf("Gateway starting on port %d", app.config.Server.Port), nil)
	return srv.ListenAndServe()
}
