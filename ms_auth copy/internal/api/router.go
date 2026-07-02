package api

import (
	"database/sql"
	"expvar"
	"ms_auth/internal/core/middleware"
	"ms_auth/internal/features/auth"
	"ms_auth/internal/features/user"
	"net/http"
)

type mw interface {
	Metrics(next http.Handler) http.Handler
	EnableCORS(next http.Handler) http.Handler
	RequireAuthenticatedUser(next http.Handler) http.Handler
	RequireActivatedUser(next http.Handler) http.Handler
	Authenticate(next http.Handler) http.Handler
	RateLimit(next http.Handler) http.Handler
	RecoverPanic(next http.Handler) http.Handler
	Logging(next http.Handler) http.Handler
	TimeoutMiddleWare(next http.Handler) http.Handler
	RequestID(next http.Handler) http.Handler
}

type errorInterceptor struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

type errorHandler interface {
	NotFoundResponse(w http.ResponseWriter, r *http.Request)
	MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request)
}

type Router struct {
	errHandler errorHandler
	m          mw
	mux        *http.ServeMux
	user       *user.UserRouter
	auth       *auth.AuthRouter
}

func NewRouter(
	handlers *handlers,
	errHandler errorHandler,
	m mw,
) *Router {
	return &Router{
		m:          m,
		errHandler: errHandler,
		mux:        http.NewServeMux(),
		user:       user.NewRouter(handlers.userHandler, m),
		auth:       auth.NewRouter(handlers.authHandler),
	}
}

func (router *Router) RegisterRoutes(db *sql.DB) http.Handler {
	mux := router.mux
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	mux.Handle("GET /metrics", middleware.MetricsHandler(db))
	mux.Handle("GET /debug/vars", expvar.Handler())

	v1Middlewares := []func(http.Handler) http.Handler{
		router.m.RateLimit,
		router.m.EnableCORS,
		// router.m.Authenticate,
	}

	v1 := http.NewServeMux()
	router.user.Routes(v1)
	router.auth.Routes(v1)

	v1Handler := Chain(v1, v1Middlewares...)
	mux.Handle("/v1/", http.StripPrefix("/v1", v1Handler))

	handler := Chain(mux,
		router.m.RecoverPanic,
		router.m.TimeoutMiddleWare,
		router.m.RequestID,
		router.m.Metrics,
		router.m.Logging,
		router.customizeErrors,
	)
	return handler
}

func (router *Router) customizeErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		interceptor := &errorInterceptor{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(interceptor, r)

		switch interceptor.status {
		case http.StatusNotFound:
			router.errHandler.NotFoundResponse(w, r)
		case http.StatusMethodNotAllowed:
			router.errHandler.MethodNotAllowedResponse(w, r)
		}
	})
}

func (i *errorInterceptor) WriteHeader(code int) {
	if i.wroteHeader {
		return
	}
	i.status = code

	if code != http.StatusNotFound && code != http.StatusMethodNotAllowed {
		i.wroteHeader = true
		i.ResponseWriter.WriteHeader(code)
	}
}

func (i *errorInterceptor) Write(b []byte) (int, error) {
	if i.status == http.StatusNotFound || i.status == http.StatusMethodNotAllowed {
		return len(b), nil
	}

	if !i.wroteHeader {
		i.WriteHeader(http.StatusOK)
	}

	return i.ResponseWriter.Write(b)
}

func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
