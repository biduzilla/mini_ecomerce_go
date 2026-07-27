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
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	tracer := otel.Tracer("ms_stock/internal/features/order")
	ctx, span := tracer.Start(r.Context(), "ProcessOrder.ProcessOrder")

	defer span.End()

	var dto OrderDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	ValidateOrderDTO(v, dto)
	if !v.Valid() {
		err := fmt.Errorf("Invalid Order DTO: %s", v.Errors)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid order DTO")
		h.errHandler.HandlerError(w, r, apiError.NewValidationError(v.Errors))
		return
	}

	orderModel := dto.ToModel()
	itemModels, err := ItemsToModels(dto.Items, orderModel.ID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to convert DTOs to Models")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	itemPtrs := make([]*OrderItem, len(itemModels))
	for i := range itemModels {
		itemPtrs[i] = &itemModels[i]
	}

	if err := h.service.processOrder(ctx, orderModel, itemPtrs); err != nil {
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
	tracer := otel.Tracer("ms_order/internal/features/order")
	ctx, span := tracer.Start(r.Context(), "GetByID.GetByID")

	defer span.End()

	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	span.SetAttributes(attribute.String("order.id", id.String()))

	orderModel, items, err := h.service.FindByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to Get by ID")
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
	tracer := otel.Tracer("ms_order/internal/features/order")
	ctx, span := tracer.Start(r.Context(), "OrderHandler.DeleteById")

	defer span.End()

	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	span.SetAttributes(attribute.String("order.id", id.String()))

	if err := h.service.DeleteById(ctx, id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusNoContent, nil, nil, h.errHandler)
}

func (h *OrderHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_order/internal/features/order")
	ctx, span := tracer.Start(r.Context(), "OrderHandler.UpdateStatus")

	defer span.End()

	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	var req struct {
		Status OrderStatus `json:"status"`
	}
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	if !IsValidOrderStatus(req.Status) {
		err := fmt.Errorf("invalid status: %s", req.Status)
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid status")
		h.errHandler.BadRequestResponse(w, r, fmt.Errorf("invalid status: %s", req.Status))
		return
	}

	span.SetAttributes(
		attribute.String("order.ID", id.String()),
		attribute.String("order.status", string(req.Status)),
	)

	if err := h.service.UpdateStatus(ctx, id, req.Status); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update status")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusNoContent, nil, nil, h.errHandler)
}
