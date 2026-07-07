package stock

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"ms_stock/internal/core/contexts"
	"ms_stock/internal/core/domain/apiError"
	"ms_stock/internal/core/jsonlog"
	"ms_stock/pkg/sqlformat"
	"strings"

	"github.com/google/uuid"
)

type StockRepository struct {
	db     *sql.DB
	logger jsonlog.Logger
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *StockRepository {
	return &StockRepository{
		db:     db,
		logger: logger,
	}
}

type repository interface {
	FindById(
		ctx context.Context,
		id uuid.UUID,
	) (*Stock, error)

	Insert(
		ctx context.Context,
		model *Stock,
	) error

	InsertAll(
		ctx context.Context,
		models []*Stock,
	) error

	Update(
		ctx context.Context,
		model *Stock,
	) error

	DeleteById(ctx context.Context, id uuid.UUID) error

	FindAllByProductIdIn(
		ctx context.Context,
		ids []uuid.UUID,
	) ([]*Stock, error)
}

func (r *StockRepository) FindById(
	ctx context.Context,
	id uuid.UUID,
) (*Stock, error) {
	query := `
	select
		s.id,
		s.product_id,
		s.available_quantity,
		s.version,
		s.created_at,
		s.created_by,
		s.updated_at,
		s.updated_by
	from stocks s
	where 
		id = $1
		and deleted = false
	`

	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)
	var model Stock
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&model.ID,
		&model.ProductId,
		&model.AvailableQuantity,
		&model.Version,
		&model.CreatedAt,
		&model.CreatedBy,
		&model.UpdatedAt,
		&model.UpdatedBy,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apiError.ErrRecordNotFound
		}
		return nil, err
	}

	return &model, nil
}

func (r *StockRepository) Insert(
	ctx context.Context,
	model *Stock,
) error {
	userAuth := contexts.GetUser(ctx)

	query := `
	insert into stocks (product_id,available_quantity,created_by)
	values (:product_id,:available_quantity,:created_by)
	returning id, created_at, version
	`

	params := map[string]any{
		"product_id":         model.ProductId,
		"available_quantity": model.AvailableQuantity,
		"created_by":         userAuth.GetID(),
	}

	query, args := sqlformat.NamedQuery(query, params)
	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	err := tx.QueryRowContext(ctx, query, args).Scan(
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

func (r *StockRepository) InsertAll(
	ctx context.Context,
	models []*Stock,
) error {
	if len(models) == 0 {
		return nil
	}

	userAuth := contexts.GetUser(ctx)
	var valueStr []string
	params := make(map[string]any)

	for i, m := range models {
		valueStr = append(
			valueStr,
			fmt.Sprintf(
				"(:product_id_%d,:available_quantity_%d,:created_by_%d)",
				i, i, i,
			),
		)

		params[fmt.Sprintf("product_id_%d", i)] = m.ProductId
		params[fmt.Sprintf("available_quantity_%d", i)] = m.AvailableQuantity
		params[fmt.Sprintf("created_by_%d", i)] = userAuth.GetID()
	}

	query := `
	insert into stocks (
		product_id,available_quantity,created_by
	) values ` +
		strings.Join(valueStr, " , ") +
		` returning id, created_at, version`

	parsedQuery, args := sqlformat.NamedQuery(query, params)
	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	rows, err := tx.QueryContext(ctx, parsedQuery, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiError.ErrEditConflict
		}
		return err
	}
	defer rows.Close()

	for i := range models {
		if rows.Next() {
			err := rows.Scan(
				&models[i].ID,
				&models[i].CreatedAt,
				&models[i].Version,
			)

			if err != nil {
				return err
			}
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return nil
}

func (r *StockRepository) Update(
	ctx context.Context,
	model *Stock,
) error {
	userAuth := contexts.GetUser(ctx)
	query := `
	update stocks
	set 
		product_id = :product_id,
		available_quantity = :available_quantity,
		updated_at = NOW(),
		updated_by = :user_id,
		version = version + 1
	WHERE 
		id = :id 
		AND version = :version 
		AND deleted = false
    RETURNING version
	`
	params := map[string]any{
		"product_id":         model.ProductId,
		"available_quantity": model.AvailableQuantity,
		"id":                 model.ID,
		"version":            model.Version,
		"user_id":            userAuth.GetID(),
	}

	query, args := sqlformat.NamedQuery(query, params)
	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	err := tx.QueryRowContext(ctx, query, args...).Scan(&model.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiError.ErrEditConflict
		}
		return err
	}

	return nil
}

func (r *StockRepository) DeleteById(ctx context.Context, id uuid.UUID) error {
	userAuth := contexts.GetUser(ctx)

	query := `
        UPDATE stocks
        SET deleted = true, updated_at = NOW(), updated_by = :user_id, version = version + 1
        WHERE id = :id AND deleted = false
        RETURNING id
    `
	params := map[string]any{
		"id":      id,
		"user_id": userAuth.GetID(),
	}

	query, args := sqlformat.NamedQuery(query, params)
	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	var returnedID uuid.UUID
	err := tx.QueryRowContext(ctx, query, args).Scan(&returnedID)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiError.ErrRecordNotFound
		}
		return err
	}

	return nil
}

func (r *StockRepository) FindAllByProductIdIn(
	ctx context.Context,
	ids []uuid.UUID,
) ([]*Stock, error) {
	if len(ids) == 0 {
		return []*Stock{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
        SELECT
            s.id,
            s.product_id,
            s.available_quantity,
            s.version,
            s.created_at,
            s.created_by,
            s.updated_at,
            s.updated_by
        FROM stocks s
        WHERE deleted = false
        AND product_id IN (%s)
    `, strings.Join(placeholders, ","))

	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	models := make([]*Stock, 0, len(ids))

	for rows.Next() {
		model := &Stock{}

		err := rows.Scan(
			&model.ID,
			&model.ProductId,
			&model.AvailableQuantity,
			&model.Version,
			&model.CreatedAt,
			&model.CreatedBy,
			&model.UpdatedAt,
			&model.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}

		models = append(models, model)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return models, nil
}
