package user

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type authMiddleware interface {
	RequireActivatedUser(next http.Handler) http.Handler
}

type UserRouter struct {
	handler userHandler
	m       authMiddleware
}

type userRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler userHandler,
	m authMiddleware,
) *UserRouter {
	return &UserRouter{
		handler: handler,
		m:       m,
	}
}

func (r *UserRouter) Routes(router chi.Router) {
	router.Route("/user", func(router chi.Router) {
		router.Post("/", r.handler.Save)

		router.Group(func(router chi.Router) {
			router.Use(r.m.RequireActivatedUser)

			// router.Get("/data", r.handler.FindAuthUserData)
			// router.Get("/{id}", r.handler.FindByID)
			// router.Put("/", r.handler.Update)
			// router.Delete("/{id}", r.handler.Delete)
		})
	})
}
