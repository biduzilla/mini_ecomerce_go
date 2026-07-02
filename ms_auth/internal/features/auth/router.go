package auth

import "net/http"

type AuthRouter struct {
	handler authHandler
}

func NewRouter(
	handler authHandler,
) *AuthRouter {
	return &AuthRouter{
		handler: handler,
	}
}

func (r *AuthRouter) Routes(mux *http.ServeMux) {
	authMux := http.NewServeMux()

	authMux.HandleFunc("POST /", r.handler.Login)
	authMux.HandleFunc("POST /auth/refresh-token", r.handler.RefreshToken)

	mux.Handle("/auth/", http.StripPrefix("/auth", authMux))
}
