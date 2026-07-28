package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ms_order/internal/core/jsonlog"
	"ms_order/internal/core/otel"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type router interface {
	RegisterRoutes(db *sql.DB) *chi.Mux
}

func (app *application) Server() error {
	defer app.db.Close()

	shutdown := make(chan struct{})

	shutdownTracer, err := otel.InitTracer("ms_order", app.Logger)
	if err != nil {
		return err
	}

	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			app.Logger.PrintError(err, nil)
		}
	}()

	deps, shutdownConsumers, err := app.buildDependencies(shutdown)
	if err != nil {
		return err
	}

	mux := deps.routers.RegisterRoutes(app.db)
	instrumentedHandler := otelhttp.NewHandler(mux, "ms_order")

	srv := newHTTPServer(fmt.Sprintf(":%d", app.config.Server.Port), instrumentedHandler, app.Logger)

	shutdownErr := make(chan error, 1)
	app.handleShutdown(srv, shutdownConsumers, shutdown, shutdownErr)

	app.Logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
	})

	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownErr
	if err != nil {
		return err
	}

	app.kafkaProducer.Close()
	app.kafkaConsumer.Close()

	app.Logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})
	return nil
}

func newHTTPServer(addr string, handler http.Handler, logger jsonlog.Logger) *http.Server {
	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ErrorLog:     log.New(logger, "", 0),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

func (app *application) handleShutdown(srv *http.Server, shutdownConsumers func(), shutdown chan struct{}, shutdownError chan<- error) {
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.Logger.PrintInfo("shutting down", map[string]string{
			"signal": s.String(),
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}

		shutdownConsumers()

		// defer app.db.Close()
		close(shutdown)

		app.Logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		app.wg.Wait()
		shutdownError <- nil
	}()
}
