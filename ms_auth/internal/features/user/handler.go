package user

import (
	"ms_auth/internal/core/handler"
	"ms_auth/pkg/httputil"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type UserHandler struct {
	service    userService
	errHandler errorHandler
}

type userHandler interface {
	// FindAuthUserData(w http.ResponseWriter, r *http.Request)
	// FindByID(w http.ResponseWriter, r *http.Request)
	Save(w http.ResponseWriter, r *http.Request)
	// Update(w http.ResponseWriter, r *http.Request)
	// Delete(w http.ResponseWriter, r *http.Request)
}

type errorHandler interface {
	HandlerError(w http.ResponseWriter, r *http.Request, err error)
	ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error)
	BadRequestResponse(w http.ResponseWriter, r *http.Request, err error)
}

func NewHandler(
	service userService,
	errHandler errorHandler,
) *UserHandler {
	return &UserHandler{
		service:    service,
		errHandler: errHandler,
	}
}

func (h *UserHandler) Save(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_auth/internal/features/user")
	ctx, span := tracer.Start(r.Context(), "UserHandler.Save")

	defer span.End()

	var dto UserDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model, err := dto.ToModel()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to transform DTO to Model")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	span.SetAttributes(attribute.String("user.email", *dto.Email))
	span.SetAttributes(attribute.String("user.Nome", *dto.Nome))

	if err := h.service.Save(ctx, model); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to save")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.ToDTO(), nil, h.errHandler)
}
