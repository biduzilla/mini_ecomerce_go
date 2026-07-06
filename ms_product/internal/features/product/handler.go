package product

import (
	"ms_product/internal/core/handler"
	"ms_product/pkg/httputil"
	"net/http"
)

type errorHandler interface {
	HandlerError(w http.ResponseWriter, r *http.Request, err error)
	ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error)
	BadRequestResponse(w http.ResponseWriter, r *http.Request, err error)
}

type ProductHandler struct {
	service    productService
	errHandler errorHandler
}

type productHandler interface {
	GetByID(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	CreateAll(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
}

func NewHandler(
	service productService,
	errHandler errorHandler,
) *ProductHandler {
	return &ProductHandler{
		service:    service,
		errHandler: errHandler,
	}
}

func (h *ProductHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	model, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(
		w, r,
		http.StatusOK,
		model.ToDTO(),
		nil,
		h.errHandler,
	)
}

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto ProductDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.ToModel()

	if err := h.service.Create(r.Context(), model); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.ToDTO(), nil, h.errHandler)
}

func (h *ProductHandler) CreateAll(w http.ResponseWriter, r *http.Request) {
	var dtos []ProductDTO
	if err := httputil.ReadJSON(w, r, &dtos); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	models := make([]*Product, len(dtos))
	for i := range dtos {
		models[i] = dtos[i].ToModel()
	}

	if err := h.service.CreateAll(r.Context(), models); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	response := make([]*ProductDTO, len(models))
	for i, m := range models {
		response[i] = m.ToDTO()
	}

	handler.Respond(w, r, http.StatusCreated, response, nil, h.errHandler)
}

func (h *ProductHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto ProductDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.ToModel()
	if err := h.service.Update(r.Context(), model); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.ToDTO(), nil, h.errHandler)
}

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(
		w,
		r,
		http.StatusNoContent,
		nil,
		nil,
		h.errHandler,
	)
}
