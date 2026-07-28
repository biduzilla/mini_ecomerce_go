package api

import (
	"ms_order/internal/core/domain/apiError"
	"ms_order/internal/core/middleware"
	"ms_order/internal/core/transaction"
)

type dependencies struct {
	repo         *repositories
	tx           transaction.Manager
	produces     *producers
	clients      *clients
	services     *services
	consumers    *consumers
	errorHandler *apiError.ErrorHandler
	handlers     *handlers
	mw           *middleware.Middleware
	routers      *Router
}

func (app *application) buildDependencies(shutdown chan struct{}) (*dependencies, func(), error) {
	repo := NewRepositories(app.db, app.Logger)
	tx := transaction.NewManager(app.db)
	producers := NewProducers(app.kafkaProducer, app.Logger)
	clients := NewClients(app.config)
	services, err := NewServices(repo, clients, producers, tx, app.config, app.Logger)
	if err != nil {
		return nil, nil, err
	}

	consumers := NewConsumers(app.kafkaConsumer, services, app.Logger)
	shutdownConsumers := consumers.Start(app.Logger)
	errHandler := apiError.NewErrorHandler(app.Logger)
	handlers := NewHandlers(services, errHandler)
	middleware := middleware.New(
		errHandler,
		app.config,
		services.jwtService,
		app.Logger,
		shutdown,
	)

	router := NewRouter(
		handlers,
		errHandler,
		middleware,
	)

	return &dependencies{
		repo:         repo,
		tx:           tx,
		produces:     producers,
		clients:      clients,
		services:     services,
		consumers:    consumers,
		errorHandler: errHandler,
		handlers:     handlers,
		mw:           middleware,
		routers:      router,
	}, shutdownConsumers, nil
}
