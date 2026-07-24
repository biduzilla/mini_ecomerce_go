package auth

import (
	"ms_auth/internal/core/handler"
	"ms_auth/pkg/httpjson"
	"ms_auth/pkg/httputil"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type errorHandler interface {
	HandlerError(w http.ResponseWriter, r *http.Request, err error)
	ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error)
	BadRequestResponse(w http.ResponseWriter, r *http.Request, err error)
}

type AuthHandler struct {
	service    authService
	errHandler errorHandler
}

type authHandler interface {
	Login(w http.ResponseWriter, r *http.Request)
	RefreshToken(w http.ResponseWriter, r *http.Request)
}

func NewHandler(
	service authService,
	errHandler errorHandler,
) *AuthHandler {
	return &AuthHandler{
		service:    service,
		errHandler: errHandler,
	}
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_auth/internal/features/auth")
	ctx, span := tracer.Start(r.Context(), "AuthHandler.Login")

	defer span.End()

	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := httputil.ReadJSON(w, r, &input)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	span.SetAttributes(attribute.String("user.email", input.Email))

	token, err := h.service.Login(ctx, input.Email, input.Password)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Authentication failed")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	span.SetStatus(codes.Ok, "Login successful")

	handler.Respond(
		w,
		r,
		http.StatusOK,
		token,
		nil,
		h.errHandler,
	)
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}

	err := httputil.ReadJSON(w, r, &input)
	if err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	accessToken, err := h.service.RefreshToken(r.Context(), input.RefreshToken)

	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(
		w,
		r,
		http.StatusOK,
		httpjson.Envelope{
			"access_token": accessToken,
		},
		nil,
		h.errHandler,
	)
}
