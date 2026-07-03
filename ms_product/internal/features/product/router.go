package product

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type authMiddleware interface {
	RequireActivatedUser(next http.Handler) http.Handler
}

type ProductRouter struct {
	handler productHandler
	m       authMiddleware
}

type productRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler productHandler,
	m authMiddleware,
) *ProductRouter {
	return &ProductRouter{
		handler: handler,
		m:       m,
	}
}

func (r *ProductRouter) Routes(router chi.Router) {
	router.Route("/product", func(router chi.Router) {
		router.Group(func(router chi.Router) {
			router.Use(r.m.RequireActivatedUser)

			router.Get("/{id}", r.handler.GetByID)
			router.Post("/", r.handler.Create)
			router.Post("/bulk", r.handler.CreateAll)
			router.Put("/", r.handler.Update)
			router.Delete("/{id}", r.handler.Delete)
		})
	})
}
