package stock

import (
	"ms_stock/internal/core/handler"
	"ms_stock/pkg/httputil"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
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
	tracer := otel.Tracer("ms_stock/internal/features/stock")
	ctx, span := tracer.Start(r.Context(), "StockHandler.GetByID")

	defer span.End()

	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	model, err := h.service.FindByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to Get by ID")
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
	tracer := otel.Tracer("ms_stock/internal/features/stock")
	ctx, span := tracer.Start(r.Context(), "StockHandler.Create")

	defer span.End()

	var dto StockDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.ToModel()

	if err := h.service.CreateStock(ctx, model); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to Create")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.ToDTO(), nil, h.errHandler)
}

func (h *StockHandler) CreateAll(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_stock/internal/features/stock")
	ctx, span := tracer.Start(r.Context(), "StockHandler.CreateAll")

	defer span.End()

	var dtos []StockDTO
	if err := httputil.ReadJSON(w, r, &dtos); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	models := make([]*Stock, len(dtos))
	for i := range dtos {
		models[i] = dtos[i].ToModel()
	}

	if err := h.service.CreateAllStock(ctx, models); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create all")
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
	tracer := otel.Tracer("ms_stock/internal/features/stock")
	ctx, span := tracer.Start(r.Context(), "StockHandler.Update")

	defer span.End()

	var dto StockDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.ToModel()
	if err := h.service.Update(ctx, model); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to update")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, model.ToDTO(), nil, h.errHandler)
}

func (h *StockHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_stock/internal/features/stock")
	ctx, span := tracer.Start(r.Context(), "StockHandler.Delete")

	defer span.End()

	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	if err := h.service.DeleteById(ctx, id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to delete")
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
	tracer := otel.Tracer("ms_stock/internal/features/stock")
	ctx, span := tracer.Start(r.Context(), "StockHandler.CheckAvailability")

	defer span.End()

	var req AvailabilityCheckRequest
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	resp, err := h.service.CheckAvailability(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to check availability")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusOK, resp, nil, h.errHandler)
}

func (h *StockHandler) DeductStock(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_stock/internal/features/stock")
	ctx, span := tracer.Start(r.Context(), "StockHandler.DeductStock")

	defer span.End()

	var req AvailabilityCheckRequest
	if err := httputil.ReadJSON(w, r, &req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	if err := h.service.DeductStock(ctx, req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to deduct stock")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusNoContent, nil, nil, h.errHandler)
}
