package order

import (
	"context"
	"fmt"
	"ms_order/internal/core/domain/apiError"
	"ms_order/internal/core/handler"
	"ms_order/internal/core/validator"
	"ms_order/pkg/httputil"
	"net/http"

	"github.com/google/uuid"
)

type errorHandler interface {
	HandlerError(w http.ResponseWriter, r *http.Request, err error)
	ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error)
	BadRequestResponse(w http.ResponseWriter, r *http.Request, err error)
}

type service interface {
	processOrder(ctx context.Context, model *Order, items []*OrderItem) error
	FindByID(ctx context.Context, id uuid.UUID) (*Order, []*OrderItem, error)
	DeleteById(ctx context.Context, id uuid.UUID) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status OrderStatus) error
}

type OrderHandler struct {
	service    service
	errHandler errorHandler
}

type orderHandler interface {
	ProcessOrder(w http.ResponseWriter, r *http.Request)
	GetByID(w http.ResponseWriter, r *http.Request)
	DeleteById(w http.ResponseWriter, r *http.Request)
	UpdateStatus(w http.ResponseWriter, r *http.Request)
}

func NewHandler(
	service service,
	errHandler errorHandler,
) *OrderHandler {
	return &OrderHandler{
		service:    service,
		errHandler: errHandler,
	}
}

func (h *OrderHandler) ProcessOrder(w http.ResponseWriter, r *http.Request) {
	var dto OrderDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	ValidateOrderDTO(v, dto)
	if !v.Valid() {
		h.errHandler.HandlerError(w, r, apiError.NewValidationError(v.Errors))
		return
	}

	orderModel := dto.ToModel()
	itemModels, err := ItemsToModels(dto.Items, orderModel.ID)
	if err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	itemPtrs := make([]*OrderItem, len(itemModels))
	for i := range itemModels {
		itemPtrs[i] = &itemModels[i]
	}

	if err := h.service.processOrder(r.Context(), orderModel, itemPtrs); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	responseDTO := orderModel.ToDTO()
	responseItems := make([]OrderItemDTO, len(itemPtrs))
	for i, item := range itemPtrs {
		responseItems[i] = *item.ToDTO()
	}
	responseDTO.Items = responseItems

	handler.Respond(w, r, http.StatusCreated, responseDTO, nil, h.errHandler)
}

func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	orderModel, items, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	responseDTO := orderModel.ToDTO()
	responseItems := make([]OrderItemDTO, len(items))
	for i, item := range items {
		responseItems[i] = *item.ToDTO()
	}
	responseDTO.Items = responseItems

	handler.Respond(w, r, http.StatusOK, responseDTO, nil, h.errHandler)
}

func (h *OrderHandler) DeleteById(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	if err := h.service.DeleteById(r.Context(), id); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusNoContent, nil, nil, h.errHandler)
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	var req struct {
		Status OrderStatus `json:"status"`
	}
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	if !IsValidOrderStatus(req.Status) {
		h.errHandler.BadRequestResponse(w, r, fmt.Errorf("invalid status: %s", req.Status))
		return
	}

	if err := h.service.UpdateStatus(r.Context(), id, req.Status); err != nil {
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusNoContent, nil, nil, h.errHandler)
}
