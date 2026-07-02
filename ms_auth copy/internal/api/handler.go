package api

import (
	"ms_auth/internal/core/domain/apiError"
	"ms_auth/internal/features/auth"
	"ms_auth/internal/features/user"
)

type handlers struct {
	userHandler *user.UserHandler
	authHandler *auth.AuthHandler
}

func NewHandlers(
	services *services,
	errHandler *apiError.ErrorHandler,
) *handlers {
	return &handlers{
		userHandler: user.NewHandler(services.userService, errHandler),
		authHandler: auth.NewHandler(services.authService, errHandler),
	}
}
