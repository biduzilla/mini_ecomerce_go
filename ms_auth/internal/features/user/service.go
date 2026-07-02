package user

import (
	"context"
	"ms_auth/internal/core/cache"
	"ms_auth/internal/core/domain/apiError"
	"ms_auth/internal/core/transaction"
	"ms_auth/internal/core/validator"
)

type UserService struct {
	repo       userRepository
	tx         transaction.Manager
	cache      cache.Cache
	keyBuilder cache.KeyBuilder
}

type userService interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	Save(ctx context.Context, model *User) error
}

func NewService(
	repo userRepository,
	tx transaction.Manager,
	cache cache.Cache,
	keyBuilder cache.KeyBuilder,
) *UserService {
	return &UserService{
		repo:       repo,
		tx:         tx,
		cache:      cache,
		keyBuilder: keyBuilder,
	}
}

func (s *UserService) FindByEmail(
	ctx context.Context,
	email string,
) (*User, error) {
	key := s.keyBuilder.BuildItemKey(email)

	return cache.FetchOrCache(ctx, s.cache, key, func() (*User, error) {
		return s.repo.FindByEmail(ctx, email)
	})
}

func (s *UserService) Save(ctx context.Context, model *User) error {
	v := validator.New()
	if model.Validate(v); !v.Valid() {
		return apiError.NewValidationError(v.Errors)
	}

	model.Roles = []Role{ROLE_CLIENT}

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
