package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func (app *application) Routes() http.Handler {
	r := chi.NewRouter()

	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Gateway OK"))
	})

	registerProxy := func(path string, targetURL, cbName, fallback string) {
		rawHandler := ProxyWithCircuitBreaker(targetURL, cbName, fallback)
		handler := otelhttp.NewHandler(rawHandler, path)

		r.Handle(path, handler)
		r.Handle(path+"/*", handler)
	}

	registerProxy("/api/auth", app.config.Services.AuthURL, "msAuthCircuitBreaker", "O serviço de autenticação está indisponível.")
	registerProxy("/api/user", app.config.Services.AuthURL, "msAuthCircuitBreaker", "O serviço de usuários está indisponível.")
	registerProxy("/api/product", app.config.Services.ProductURL, "msProductCircuitBreaker", "O serviço de produtos está indisponível.")
	registerProxy("/api/stock", app.config.Services.StockURL, "msStockCircuitBreaker", "O serviço de estoque está temporariamente indisponível.")
	registerProxy("/api/orders", app.config.Services.OrderURL, "msOrderCircuitBreaker", "O serviço de pedidos está temporariamente indisponível.")

	return r
}
