package stock

import (
	"context"
	"errors"
	"fmt"
	"ms_stock/internal/core/cache"
	"ms_stock/internal/core/clients/product"
	"ms_stock/internal/core/domain/apiError"
	"ms_stock/internal/core/transaction"
	"ms_stock/internal/core/validator"
	"net/http"

	"github.com/google/uuid"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock for this product")
)

type StockService struct {
	repo          repository
	tx            transaction.Manager
	cache         cache.Cache
	keyBuilder    cache.KeyBuilder
	productClient product.Client
}

func NewService(
	repo repository,
	tx transaction.Manager,
	cache cache.Cache,
	keyBuilder cache.KeyBuilder,
	productClient product.Client,
) *StockService {
	return &StockService{
		repo:          repo,
		tx:            tx,
		cache:         cache,
		keyBuilder:    keyBuilder,
		productClient: productClient,
	}
}

func (s *StockService) FindByID(
	ctx context.Context,
	id uuid.UUID,
) (*Stock, error) {
	key := s.keyBuilder.BuildItemKey(id.String())
	return cache.FetchOrCache(ctx, s.cache, key, func() (*Stock, error) {
		return s.repo.FindById(ctx, id)
	})
}

func (s *StockService) Insert(
	ctx context.Context,
	model *Stock,
) error {
	v := validator.New()
	if model.Validate(v); !v.Valid() {
		return apiError.NewValidationError(v.Errors)
	}

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.Insert(ctx, model)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(ctx, s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *StockService) InsertAll(
	ctx context.Context,
	models []*Stock,
) error {
	v := validator.New()
	for _, m := range models {
		if m.Validate(v); !v.Valid() {
			return apiError.NewValidationError(v.Errors)
		}
	}

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.InsertAll(ctx, models)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(ctx, s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *StockService) CreateAStock(
	ctx context.Context,
	model *Stock,
) error {
	_, err := s.productClient.GetByID(ctx, model.ProductId)
	if err != nil {
		return err
	}

	return s.Insert(ctx, model)
}

func (s *StockService) CreateAllStock(
	ctx context.Context,
	models []*Stock,
) error {
	for _, m := range models {
		_, err := s.productClient.GetByID(ctx, m.ProductId)
		if err != nil {
			return err
		}
	}

	return s.InsertAll(ctx, models)
}

func (s *StockService) DeleteById(
	ctx context.Context,
	id uuid.UUID,
) error {
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.DeleteById(ctx, id)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(ctx, s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *StockService) FindAllByProductIn(
	ctx context.Context,
	ids []uuid.UUID,
) ([]*Stock, error) {
	key := s.keyBuilder.BuildListKey(ids)
	return cache.FetchOrCache(ctx, s.cache, key, func() ([]*Stock, error) {
		return s.repo.FindAllByProductIdIn(ctx, ids)
	})
}

func (s *StockService) CheckAvailability(
	ctx context.Context,
	request AvailabilityCheckRequest,
) (*AvailabilityCheckResponse, error) {
	v := validator.New()
	if request.Validate(v); !v.Valid() {
		return nil, apiError.NewValidationError(v.Errors)
	}

	stockMap, err := s.getStockMap(ctx, request.Items)
	if err != nil {
		return nil, err
	}

	var details []ItemAvailabilityDetail

	for _, item := range request.Items {
		available := 0
		if stock, exists := stockMap[item.ProductID]; exists {
			available = stock.AvailableQuantity
		}

		if item.Quantity > available {
			details = append(details, ItemAvailabilityDetail{
				ProductID: item.ProductID,
				Requested: item.Quantity,
				Available: available,
			})
		}
	}

	return &AvailabilityCheckResponse{
		Available: len(details) == 0,
		Details:   details,
	}, nil
}

func (s *StockService) DeductStock(
	ctx context.Context,
	req AvailabilityCheckRequest,
) error {
	stockMap, err := s.getStockMap(ctx, req.Items)
	if err != nil {
		return err
	}

	err = s.tx.RunInTx(ctx, func(ctx context.Context) error {
		for _, item := range req.Items {
			stock, exists := stockMap[item.ProductID]

			if !exists {
				return apiError.NewApiError(
					fmt.Sprintf("%s: product %s", apiError.ErrRecordNotFound, item.ProductID),
					http.StatusConflict,
				)
			}

			if stock.AvailableQuantity < item.Quantity {
				return apiError.NewApiError(
					fmt.Sprintf("%s for product %s", ErrInsufficientStock, item.ProductID),
					http.StatusConflict,
				)
			}

			stock.AvailableQuantity -= item.Quantity

			if err := s.repo.Update(ctx, stock); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(ctx, s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *StockService) getStockMap(
	ctx context.Context,
	items []ItemRequest) (map[uuid.UUID]*Stock, error) {
	productIds := make([]uuid.UUID, len(items))
	for i, item := range items {
		productIds[i] = item.ProductID
	}

	stocks, err := s.FindAllByProductIn(ctx, productIds)
	if err != nil {
		return nil, err
	}

	stockMap := make(map[uuid.UUID]*Stock, len(stocks))
	for _, stock := range stocks {
		stockMap[stock.ProductId] = stock
	}

	return stockMap, nil
}
