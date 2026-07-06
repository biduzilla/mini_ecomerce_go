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

	"ms_stock/internal/core/domain/apiError"
	"ms_stock/internal/core/middleware"
	"ms_stock/internal/core/transaction"

	"github.com/go-chi/chi/v5"
)

type router interface {
	RegisterRoutes(db *sql.DB) *chi.Mux
}

func (app *application) Server() error {
	defer app.db.Close()

	shutdown := make(chan struct{})

	repo := NewRepositories(app.db, app.Logger)
	tx := transaction.NewManager(app.db)
	services, err := NewServices(repo, tx, app.config, app.Logger)
	if err != nil {
		return err
	}

	errHandler := apiError.NewErrorHandler(app.Logger)
	handlers := NewHandlers(services, errHandler)
	middleware := middleware.New(
		errHandler,
		app.config,
		services.jwtService,
		app.Logger,
		shutdown,
	)

	var router router = NewRouter(
		handlers,
		errHandler,
		middleware,
	)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.config.Server.Port),
		Handler:      router.RegisterRoutes(app.db),
		IdleTimeout:  time.Minute,
		ErrorLog:     log.New(app.Logger, "", 0),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	shutdownError := make(chan error)

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
