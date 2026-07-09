package order

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type authMiddleware interface {
	RequireActivatedUser(next http.Handler) http.Handler
}

type OrderRouter struct {
	handler orderHandler
	m       authMiddleware
}

type orderRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler orderHandler,
	m authMiddleware,
) *OrderRouter {
	return &OrderRouter{
		handler: handler,
		m:       m,
	}
}

func (r *OrderRouter) Routes(router chi.Router) {
	router.Route("/orders", func(router chi.Router) {
		router.Group(func(router chi.Router) {
			router.Use(r.m.RequireActivatedUser)

			router.Post("/", r.handler.ProcessOrder)
			router.Get("/{id}", r.handler.GetByID)
			router.Delete("/{id}", r.handler.DeleteById)
			router.Patch("/{id}/status", r.handler.UpdateStatus)
		})
	})
}
