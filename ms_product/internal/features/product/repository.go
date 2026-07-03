package product

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"ms_product/internal/core/contexts"
	"ms_product/internal/core/domain/apiError"
	"ms_product/internal/core/jsonlog"
	"ms_product/pkg/sqlformat"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ProductRepository struct {
	db     *sql.DB
	logger jsonlog.Logger
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *ProductRepository {
	return &ProductRepository{
		db:     db,
		logger: logger,
	}
}

func parseProductsConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "uniq_products_name":
			return apiError.ValidationAlreadyExists("name")
		}
		return err
	}

	return nil
}

type productRepository interface {
	Insert(ctx context.Context, model *Product) error
	InsertAll(
		ctx context.Context,
		models []*Product,
	) error
	GetByID(ctx context.Context, id uuid.UUID) (*Product, error)
	Update(ctx context.Context, model *Product) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func (r *ProductRepository) Insert(ctx context.Context, model *Product) error {
	userAuth := contexts.GetUser(ctx)
	query := `
        INSERT INTO products (name, price, created_by)
        VALUES (:name, :price, :created_by)
        RETURNING id, created_at, version
    `
	params := map[string]any{
		"name":       model.Name,
		"price":      model.Price,
		"created_by": userAuth.GetID(),
	}

	query, args := sqlformat.NamedQuery(query, params)
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
		return parseProductsConstraintError(err)
	}

	return nil
}

func (r *ProductRepository) InsertAll(
	ctx context.Context,
	models []*Product,
) error {
	if len(models) == 0 {
		return nil
	}

	userAuth := contexts.GetUser(ctx)
	var valueStrings []string
	params := make(map[string]any)

	for i, m := range models {
		valueStrings = append(
			valueStrings,
			fmt.Sprintf("(:name_%d, :price:_%d, :created_by_%d)", i, i, i),
		)

		params[fmt.Sprintf("name_%d", i)] = m.Name
		params[fmt.Sprintf("price_%d", i)] = m.Price
		params[fmt.Sprintf("created_by_%d", i)] = userAuth.GetID()
	}

	query := `INSERT INTO PRODUCTS (
		name, price, created_by
	) VALUES ` +
		strings.Join(valueStrings, " , ") +
		` RETURNING ID, created_at , version`

	parsedQuery, args := sqlformat.NamedQuery(query, params)
	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	rows, err := tx.QueryContext(ctx, parsedQuery, args...)
	if err != nil {
		return parseProductsConstraintError(err)
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

func (r *ProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	query := `
        SELECT 
            id,
			name,
			price,
			version,
			created_at,
			created_by,
			updated_at,
			updated_by
        FROM products
        WHERE id = $1 AND deleted = false
    `
	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	var model Product
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&model.ID,
		&model.Name,
		&model.Price,
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

func (r *ProductRepository) Update(ctx context.Context, model *Product) error {
	userAuth := contexts.GetUser(ctx)

	query := `
        UPDATE products
        SET 
			name = :name,
			price = :price,
            updated_at = NOW(),
			updated_by = :user_id,
			version = version + 1
        WHERE id = :id AND version = :version AND deleted = false
        RETURNING version
    `
	params := map[string]any{
		"name":    model.Name,
		"price":   model.Price,
		"id":      model.ID,
		"version": model.Version,
		"user_id": userAuth.GetID(),
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
		return parseProductsConstraintError(err)
	}

	return nil
}

func (r *ProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
	userAuth := contexts.GetUser(ctx)

	query := `
        UPDATE products
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
