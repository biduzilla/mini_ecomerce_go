package stock

import (
	"ms_stock/internal/core/handler"
	"ms_stock/pkg/httputil"
	"net/http"
)

type errorHandler interface {
	HandlerError(w http.ResponseWriter, r *http.Request, err error)
	ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error)
	BadRequestResponse(w http.ResponseWriter, r *http.Request, err error)
}

type StockHandler struct {
	service    service
	errHandler errorHandler
}

type stockHandler interface {
	GetByID(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	CreateAll(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
	CheckAvailability(w http.ResponseWriter, r *http.Request)
	DeductStock(w http.ResponseWriter, r *http.Request)
}

func NewHandler(
	service service,
	errHandler errorHandler,
) *StockHandler {
	return &StockHandler{
		service:    service,
		errHandler: errHandler,
	}
}

func (h *StockHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	model, err := h.service.FindByID(r.Context(), id)
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

func (h *StockHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dto StockDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.ToModel()

	if err := h.service.CreateStock(r.Context(), model); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.ToDTO(), nil, h.errHandler)
}

func (h *StockHandler) CreateAll(w http.ResponseWriter, r *http.Request) {
	var dtos []StockDTO
	if err := httputil.ReadJSON(w, r, &dtos); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	models := make([]*Stock, len(dtos))
	for i := range dtos {
		models[i] = dtos[i].ToModel()
	}

	if err := h.service.CreateAllStock(r.Context(), models); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	response := make([]*StockDTO, len(models))
	for i, m := range models {
		response[i] = m.ToDTO()
	}

	handler.Respond(w, r, http.StatusCreated, response, nil, h.errHandler)
}

func (h *StockHandler) Update(w http.ResponseWriter, r *http.Request) {
	var dto StockDTO
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

func (h *StockHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	if err := h.service.DeleteById(r.Context(), id); err != nil {
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

func (h *StockHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	var req AvailabilityCheckRequest
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	resp, err := h.service.CheckAvailability(r.Context(), req)
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, resp, nil, h.errHandler)
}

func (h *StockHandler) DeductStock(w http.ResponseWriter, r *http.Request) {
	var req AvailabilityCheckRequest
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	if err := h.service.DeductStock(r.Context(), req); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusNoContent, nil, nil, h.errHandler)
}
