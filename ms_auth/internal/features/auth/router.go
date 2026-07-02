package auth

import (
	"github.com/go-chi/chi/v5"
)

type AuthRouter struct {
	handler authHandler
}

type authRouter interface {
	Routes(router chi.Router)
}

func NewRouter(
	handler authHandler,
) *AuthRouter {
	return &AuthRouter{
		handler: handler,
	}
}

func (r *AuthRouter) Routes(router chi.Router) {
	router.Route("/auth", func(router chi.Router) {
		router.Post("/", r.handler.Login)
		router.Post("/refresh-token", r.handler.RefreshToken)
	})
}
