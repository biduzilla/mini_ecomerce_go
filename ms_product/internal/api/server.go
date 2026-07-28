package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"ms_product/internal/core/jsonlog"
	"ms_product/internal/core/otel"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type router interface {
	RegisterRoutes(db *sql.DB) *chi.Mux
}

func (app *application) Server() error {
	defer app.db.Close()

	shutdown := make(chan struct{})

	shutdownTracer, err := otel.InitTracer("ms_product", app.Logger)
	if err != nil {
		return err
	}

	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			app.Logger.PrintError(err, nil)
		}
	}()

	deps, err := app.buildDependencies(shutdown)
	if err != nil {
		return nil
	}

	mux := deps.routers.RegisterRoutes(app.db)
	instrumentedHandler := otelhttp.NewHandler(mux, "ms_product")

	srv := newHTTPServer(
		fmt.Sprintf(":%d", app.config.Server.Port),
		instrumentedHandler,
		app.Logger,
	)

	shutdownError := make(chan error, 1)
	app.handleShutdown(srv, shutdown, shutdownError)

	app.Logger.PrintInfo("starting server", map[string]string{
		"addr": srv.Addr,
	})

	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

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

func (app *application) handleShutdown(srv *http.Server, shutdown chan struct{}, shutdownError chan<- error) {
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

		// defer app.db.Close()
		close(shutdown)

		app.Logger.PrintInfo("completing background tasks", map[string]string{
			"addr": srv.Addr,
		})

		app.wg.Wait()
		shutdownError <- nil
	}()
}
