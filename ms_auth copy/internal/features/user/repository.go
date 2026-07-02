package user

import (
	"context"
	"database/sql"
	"errors"
	"ms_auth/internal/core/contexts"
	"ms_auth/internal/core/domain/apiError"
	"ms_auth/internal/core/jsonlog"
	"ms_auth/pkg/sqlformat"

	"github.com/lib/pq"
)

type UserRepository struct {
	db     *sql.DB
	logger jsonlog.Logger
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *UserRepository {
	return &UserRepository{
		db:     db,
		logger: logger,
	}
}

type userRepository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	Insert(ctx context.Context, model *User) error
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	query := `
        SELECT 
            id,
            nome,
            email,
            password_hash,
            activated,
            roles
        FROM users
        WHERE email = $1
			and deleted = false
    `
	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	var model User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&model.ID,
		&model.Nome,
		&model.Email,
		&model.Senha.Hash,
		&model.Activated,
		pq.Array(&model.Roles),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apiError.ErrRecordNotFound
		}
		return nil, err
	}

	return &model, nil
}

func (r *UserRepository) Insert(ctx context.Context, model *User) error {
	query := `
	INSERT INTO users (name, email, password_hash,roles, activated,deleted)
	VALUES ($1, $2, $3, $4, $5, $6,false)
	RETURNING id, created_at, version
	`
	args := []any{
		model.Nome,
		model.Email,
		model.Senha.Hash,
		model.Roles,
		model.Activated,
	}

	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	err := tx.QueryRowContext(ctx, query, args...).Scan(
		&model.ID,
		&model.CreatedAt,
		&model.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiError.ErrEditConflict
		}
		return err
	}

	return nil
}
