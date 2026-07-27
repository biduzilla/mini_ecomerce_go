package product

import (
	"ms_product/internal/core/handler"
	"ms_product/pkg/httputil"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
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
	tracer := otel.Tracer("ms_product/internal/features/product")
	ctx, span := tracer.Start(r.Context(), "ProductHandler.GetByID")

	defer span.End()

	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	model, err := h.service.GetByID(ctx, id)
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

func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_product/internal/features/product")
	ctx, span := tracer.Start(r.Context(), "ProductHandler.Create")

	defer span.End()

	var dto ProductDTO
	if err := httputil.ReadJSON(w, r, &dto); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	model := dto.ToModel()

	if err := h.service.Create(ctx, model); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to Create")
		h.errHandler.HandlerError(w, r, err)
		return
	}

	handler.Respond(w, r, http.StatusCreated, model.ToDTO(), nil, h.errHandler)
}

func (h *ProductHandler) CreateAll(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_product/internal/features/product")
	ctx, span := tracer.Start(r.Context(), "ProductHandler.CreateAll")

	defer span.End()

	var dtos []ProductDTO
	if err := httputil.ReadJSON(w, r, &dtos); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to read JSON")
		h.errHandler.BadRequestResponse(w, r, err)
		return
	}

	models := make([]*Product, len(dtos))
	for i := range dtos {
		models[i] = dtos[i].ToModel()
	}

	if err := h.service.CreateAll(ctx, models); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Failed to create all")
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
	tracer := otel.Tracer("ms_auth/internal/features/product")
	ctx, span := tracer.Start(r.Context(), "ProductHandler.Update")

	defer span.End()

	var dto ProductDTO
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

func (h *ProductHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tracer := otel.Tracer("ms_auth/internal/features/product")
	ctx, span := tracer.Start(r.Context(), "AuthHandler.Delete")

	defer span.End()

	id, ok := handler.ParseUUID(w, r, h.errHandler)
	if !ok {
		return
	}

	if err := h.service.Delete(ctx, id); err != nil {
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
