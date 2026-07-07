package stock

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type authMiddleware interface {
	RequireActivatedUser(next http.Handler) http.Handler
}

type StockRouter struct {
	handler stockHandler
	m       authMiddleware
}

type stockRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler stockHandler,
	m authMiddleware,
) *StockRouter {
	return &StockRouter{
		handler: handler,
		m:       m,
	}
}

func (r *StockRouter) Routes(router chi.Router) {
	router.Route("/stock", func(router chi.Router) {
		router.Group(func(router chi.Router) {
			router.Use(r.m.RequireActivatedUser)

			router.Get("/{id}", r.handler.GetByID)
			router.Post("/", r.handler.Create)
			router.Post("/bulk", r.handler.CreateAll)
			router.Put("/", r.handler.Update)
			router.Delete("/{id}", r.handler.Delete)

			router.Post("/check-availability", r.handler.CheckAvailability)
			router.Post("/deduct", r.handler.DeductStock)
		})
	})
}
