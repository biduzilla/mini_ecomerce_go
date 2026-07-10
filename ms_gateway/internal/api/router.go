package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) Routes() http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Gateway OK"))
	})

	r.Handle("/api/auth/*", ProxyWithCircuitBreaker(
		app.config.Services.AuthURL,
		"msAuthCircuitBreaker",
		"O serviço de autenticação está indisponível no momento.",
		app.logger,
	))

	r.Handle("/api/products/*", ProxyWithCircuitBreaker(
		app.config.Services.ProductURL,
		"msProductCircuitBreaker",
		"O serviço de produtos está indisponível no momento.",
		app.logger,
	))

	r.Handle("/api/stocks/*", ProxyWithCircuitBreaker(
		app.config.Services.StockURL,
		"msStockCircuitBreaker",
		"O serviço de estoque está temporariamente indisponível.",
		app.logger,
	))

	r.Handle("/api/orders/*", ProxyWithCircuitBreaker(
		app.config.Services.OrderURL,
		"msOrderCircuitBreaker",
		"O serviço de pedidos está temporariamente indisponível.",
		app.logger,
	))

	return r
}
