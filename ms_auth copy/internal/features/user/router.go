package user

import "net/http"

type authMiddleware interface {
	RequireActivatedUser(next http.Handler) http.Handler
}

type userHandler interface {
	Save(w http.ResponseWriter, r *http.Request)
}

type UserRouter struct {
	handler userHandler
	m       authMiddleware
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

func (r *UserRouter) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /user/", r.handler.Save)
}
