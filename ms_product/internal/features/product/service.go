package product

import (
	"context"
	"ms_product/internal/core/cache"
	"ms_product/internal/core/domain/apiError"
	"ms_product/internal/core/transaction"
	"ms_product/internal/core/validator"

	"github.com/google/uuid"
)

type ProductService struct {
	repo       productRepository
	tx         transaction.Manager
	cache      cache.Cache
	keyBuilder cache.KeyBuilder
}

type productService interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	Create(ctx context.Context, model *Product) error
	CreateAll(ctx context.Context, models []*Product) error
	Update(ctx context.Context, model *Product) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewService(
	repo productRepository,
	tx transaction.Manager,
	cache cache.Cache,
	keyBuilder cache.KeyBuilder,
) *ProductService {
	return &ProductService{
		repo:       repo,
		tx:         tx,
		cache:      cache,
		keyBuilder: keyBuilder,
	}
}

func (s *ProductService) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (*Product, error) {
	key := s.keyBuilder.BuildItemKey(id.String())

	return cache.FetchOrCache(ctx, s.cache, key, func() (*Product, error) {
		return s.repo.GetByID(ctx, id)
	})
}

func (s *ProductService) Create(ctx context.Context, model *Product) error {
	v := validator.New()
	if model.Validate(v); !v.Valid() {
		return apiError.NewValidationError(v.Errors)
	}

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.repo.Insert(ctx, model)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(context.Background(), s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *ProductService) CreateAll(ctx context.Context, models []*Product) error {
	for _, m := range models {
		v := validator.New()
		if m.Validate(v); !v.Valid() {
			return apiError.NewValidationError(v.Errors)
		}
	}

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.repo.InsertAll(ctx, models)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(context.Background(), s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *ProductService) Update(ctx context.Context, model *Product) error {
	v := validator.New()
	if model.Validate(v); !v.Valid() {
		return apiError.NewValidationError(v.Errors)
	}

	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.repo.Update(ctx, model)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(context.Background(), s.keyBuilder.GetPrefix())
	}()

	return nil
}

func (s *ProductService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.tx.RunInTx(ctx, func(ctx context.Context) error {
		return s.repo.Delete(ctx, id)
	})

	if err != nil {
		return err
	}

	go func() {
		_ = s.cache.DeleteByPrefix(context.Background(), s.keyBuilder.GetPrefix())
	}()

	return nil
}
