package api

import (
	"context"
	"fmt"
	"ms_gateway/internal/core/jsonlog"
	"ms_gateway/internal/core/jsonlog/otel"
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

	shutdownTracer, err := otel.InitTracer("ms_gateway", app.logger)
	if err != nil {
		return err
	}

	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			app.logger.PrintError(err, nil)
		}
	}()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", app.config.Server.Port),
		Handler: app.Routes(),
	}

	app.logger.PrintInfo(fmt.Sprintf("Gateway starting on port %d", app.config.Server.Port), nil)
	return srv.ListenAndServe()
}
