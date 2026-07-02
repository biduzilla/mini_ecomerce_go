package user

import (
	"ms_auth/internal/core/handler"
	"ms_auth/pkg/httputil"
	"net/http"
)

type UserHandler struct {
	service    userService
	errHandler errorHandler
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
	var dto UserDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model, err := dto.ToModel()
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	if err := h.service.Save(r.Context(), model); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.ToDTO(), nil, h.errHandler)
}
