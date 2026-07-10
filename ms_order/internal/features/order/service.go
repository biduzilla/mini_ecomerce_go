package order

import (
	"context"
	"fmt"
	"ms_order/internal/core/cache"
	"ms_order/internal/core/clients/stock"
	"ms_order/internal/core/domain/apiError"
	"ms_order/internal/core/events"
	"ms_order/internal/core/jsonlog"
	"ms_order/internal/core/transaction"
	"ms_order/internal/core/validator"
	"net/http"

	"github.com/google/uuid"
)

type OrderService struct {
	repo        repository
	tx          transaction.Manager
	cache       cache.Cache
	keyBuilder  cache.KeyBuilder
	stockClient stock.Client
	orderProducer
	logger jsonlog.Logger
}

type orderProducer interface {
	PublishOrderCreated(
		ctx context.Context,
		event *events.OrderCreatedEvent,
	) error
}

func NewService(
	repo repository,
	tx transaction.Manager,
	cache cache.Cache,
	keyBuilder cache.KeyBuilder,
	stockClient stock.Client,
	orderProducer orderProducer,
	logger jsonlog.Logger,
) *OrderService {
	return &OrderService{
		repo:          repo,
		tx:            tx,
		cache:         cache,
		keyBuilder:    keyBuilder,
		stockClient:   stockClient,
		orderProducer: orderProducer,
		logger:        logger,
	}
}

func (s *OrderService) Create(
	ctx context.Context,
	model *Order,
	items []*OrderItem,
) error {
	v := validator.New()
	if model.Validate(v); !v.Valid() {
		return apiError.NewValidationError(v.Errors)
	}

	model.TotalAmount = CalculateTotalFromItems(items)

	for _, item := range items {
		if item.Validate(v); !v.Valid() {
			return apiError.NewValidationError(v.Errors)
		}
	}

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.repo.InsertWithItems(ctx, model, items)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(ctx, s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *OrderService) checkAvailability(
	ctx context.Context,
	model *Order,
	items []*OrderItem,
) error {
	v := validator.New()
	if model.Validate(v); !v.Valid() {
		return apiError.NewValidationError(v.Errors)
	}

	for _, item := range items {
		if item.Validate(v); !v.Valid() {
			return apiError.NewValidationError(v.Errors)
		}
	}

	itensRequest := make([]stock.ItemRequest, len(items))
	for i, item := range items {
		itensRequest[i] = stock.ItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	availabilityRequest := stock.AvailabilityCheckRequest{
		Items: itensRequest,
	}

	response, err := s.stockClient.CheckAvailability(ctx, availabilityRequest)
	if err != nil {
		return err
	}

	if !response.Available {
		details := make(map[string]string)
		for _, d := range response.Details {
			details["error"] = fmt.Sprintf("product_id %d, requested %d, available %d", d.ProductID, d.Requested, d.Available)
		}

		return apiError.NewDetailedApiError(
			"insufficient stock for one or more products",
			http.StatusUnprocessableEntity,
			details,
		)
	}

	return nil
}

func (s *OrderService) buildOrderCreatedEvent(
	model *Order,
	items []*OrderItem,
) *events.OrderCreatedEvent {
	itemsEvent := make([]events.OrderItemEvent, len(items))
	for i, item := range items {
		itemsEvent[i] = events.OrderItemEvent{
			ProductID:  item.ProductID,
			ID:         item.ID,
			Quantity:   item.Quantity,
			UnitPrice:  item.UnitPrice,
			TotalPrice: item.UnitPrice * float64(item.Quantity),
		}
	}

	return events.NewOrderCreatedEvent(
		model.ID,
		string(model.Status),
		model.TotalAmount,
		itemsEvent,
	)
}

func (s *OrderService) processOrder(
	ctx context.Context,
	model *Order,
	items []*OrderItem,
) error {
	if err := s.checkAvailability(ctx, model, items); err != nil {
		return err
	}

	err := s.Create(ctx, model, items)
	if err != nil {
		return err
	}

	err = s.orderProducer.PublishOrderCreated(
		ctx,
		s.buildOrderCreatedEvent(model, items),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s *OrderService) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*Order, []*OrderItem, error) {
	key := s.keyBuilder.BuildItemKey(id.String())

	type orderPayload struct {
		Order *Order
		Items []*OrderItem
	}

	payload, err := cache.FetchOrCache(ctx, s.cache, key, func() (*orderPayload, error) {
		order, items, err := s.repo.FindByIdWithItems(ctx, id)
		if err != nil {
			return nil, err
		}
		return &orderPayload{
			Order: order,
			Items: items,
		}, nil
	})
	if err != nil {
		return nil, nil, err
	}

	return payload.Order, payload.Items, nil
}

func (s *OrderService) DeleteById(
	ctx context.Context,
	id uuid.UUID,
) error {
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.repo.DeleteById(ctx, id)
	})
	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(ctx, s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *OrderService) UpdateStatus(
	ctx context.Context,
	id uuid.UUID,
	status OrderStatus,
) error {
	order, _, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}
	order.Status = status

	err = s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.repo.Update(ctx, order)
	})
	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(ctx, s.keyBuilder.GetPrefix())
	}()

	return nil
}
