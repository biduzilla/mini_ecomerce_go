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

	"ms_order/internal/core/domain/apiError"
	"ms_order/internal/core/middleware"
	"ms_order/internal/core/transaction"

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
	producers := NewProducers(app.kafkaProducer, app.Logger)
	clients := NewClients(app.config)
	services, err := NewServices(repo, clients, producers, tx, app.config, app.Logger)
	if err != nil {
		return err
	}

	consumers := NewConsumers(app.kafkaConsumer, services, app.Logger)
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

	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		consumerCtx, cancel := context.WithCancel(context.Background())

		go func() {
			<-shutdown
			cancel()
		}()

		app.Logger.PrintInfo("starting kafka order consumer...", nil)

		if err := consumers.stockConsumers.Start(consumerCtx); err != nil {
			app.Logger.PrintError(fmt.Errorf("kafka consumer error: %w", err), nil)
		}

		app.Logger.PrintInfo("kafka order consumer stopped gracefully", nil)
	}()

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

	app.kafkaProducer.Close()
	app.kafkaConsumer.Close()

	app.Logger.PrintInfo("stopped server", map[string]string{
		"addr": srv.Addr,
	})
	return nil
}
