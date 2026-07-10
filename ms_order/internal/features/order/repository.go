package order

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"ms_order/internal/core/contexts"
	"ms_order/internal/core/domain/apiError"
	"ms_order/internal/core/jsonlog"
	"ms_order/pkg/sqlformat"
	"strings"

	"github.com/google/uuid"
)

type OrderRepository struct {
	db     *sql.DB
	logger jsonlog.Logger
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) *OrderRepository {
	return &OrderRepository{
		db:     db,
		logger: logger,
	}
}

type repository interface {
	InsertWithItems(
		ctx context.Context,
		order *Order,
		items []*OrderItem,
	) error

	FindById(
		ctx context.Context,
		id uuid.UUID,
	) (*Order, error)

	FindByIdWithItems(
		ctx context.Context,
		id uuid.UUID,
	) (*Order, []*OrderItem, error)

	FindItemsByOrderId(
		ctx context.Context,
		orderId uuid.UUID,
	) ([]*OrderItem, error)

	Update(
		ctx context.Context,
		model *Order,
	) error

	DeleteById(
		ctx context.Context,
		id uuid.UUID,
	) error
}

func (r *OrderRepository) InsertWithItems(
	ctx context.Context,
	order *Order,
	items []*OrderItem,
) error {
	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	userAuth := contexts.GetUser(ctx)

	orderQuery := `
    INSERT INTO orders (total_amount, status, created_by)
    VALUES (:total_amount, :status, :created_by)
    RETURNING id, created_at, version
    `

	orderParams := map[string]any{
		"total_amount": order.TotalAmount,
		"status":       order.Status,
		"created_by":   userAuth.GetID(),
	}

	parsedQuery, args := sqlformat.NamedQuery(orderQuery, orderParams)
	r.logger.PrintInfo(sqlformat.MinifySQL(parsedQuery), nil)

	err := tx.QueryRowContext(ctx, parsedQuery, args...).Scan(
		&order.ID,
		&order.CreatedAt,
		&order.Version,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiError.ErrEditConflict
		}
		return err
	}

	if len(items) > 0 {
		for _, item := range items {
			item.OrderID = order.ID
		}

		err := r.insertItems(ctx, items)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *OrderRepository) insertItems(
	ctx context.Context,
	items []*OrderItem,
) error {
	if len(items) == 0 {
		return nil
	}

	userAuth := contexts.GetUser(ctx)
	var valueStr []string
	params := make(map[string]any)

	for i, m := range items {
		valueStr = append(
			valueStr,
			fmt.Sprintf(
				"(:order_id_%d,:product_id_%d,:quantity_%d,:unit_price_%d,:created_by_%d)",
				i, i, i, i, i,
			),
		)

		params[fmt.Sprintf("order_id_%d", i)] = m.OrderID
		params[fmt.Sprintf("product_id_%d", i)] = m.ProductID
		params[fmt.Sprintf("quantity_%d", i)] = m.Quantity
		params[fmt.Sprintf("unit_price_%d", i)] = m.UnitPrice
		params[fmt.Sprintf("created_by_%d", i)] = userAuth.GetID()
	}

	query := `
	insert into order_items (
		order_id, product_id, quantity,unit_price, created_by
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

	for i := range items {
		if rows.Next() {
			err := rows.Scan(
				&items[i].ID,
				&items[i].CreatedAt,
				&items[i].Version,
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

func (r *OrderRepository) FindById(
	ctx context.Context,
	id uuid.UUID,
) (*Order, error) {
	query := `
        SELECT
            o.id,
            o.total_amount,
            o.status,
            o.version,
            o.created_at,
            o.created_by,
            o.updated_at,
            o.updated_by
        FROM orders o
        WHERE o.id = $1
            AND o.deleted = false
    `

	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	var model Order
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&model.ID,
		&model.TotalAmount,
		&model.Status,
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

func (r *OrderRepository) FindByIdWithItems(
	ctx context.Context,
	id uuid.UUID,
) (*Order, []*OrderItem, error) {
	order, err := r.FindById(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	items, err := r.FindItemsByOrderId(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return order, items, nil
}

func (r *OrderRepository) FindItemsByOrderId(
	ctx context.Context,
	orderId uuid.UUID,
) ([]*OrderItem, error) {
	query := `
        SELECT
            oi.id,
            oi.order_id,
            oi.product_id,
            oi.quantity,
            oi.unit_price,
            oi.version,
            oi.created_at,
            oi.created_by,
            oi.updated_at,
            oi.updated_by
        FROM order_items oi
        WHERE oi.order_id = $1
            AND oi.deleted = false
        ORDER BY oi.created_at ASC
    `

	r.logger.PrintInfo(sqlformat.MinifySQL(query), nil)

	rows, err := r.db.QueryContext(ctx, query, orderId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*OrderItem, 0)

	for rows.Next() {
		item := &OrderItem{}

		err := rows.Scan(
			&item.ID,
			&item.OrderID,
			&item.ProductID,
			&item.Quantity,
			&item.UnitPrice,
			&item.Version,
			&item.CreatedAt,
			&item.CreatedBy,
			&item.UpdatedAt,
			&item.UpdatedBy,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *OrderRepository) Update(
	ctx context.Context,
	model *Order,
) error {
	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	userAuth := contexts.GetUser(ctx)

	query := `
        UPDATE orders
        SET
            total_amount = :total_amount,
            status = :status,
            updated_at = NOW(),
            updated_by = :user_id,
            version = version + 1
        WHERE id = :id
            AND version = :version
            AND deleted = false
        RETURNING version
    `

	params := map[string]any{
		"total_amount": model.TotalAmount,
		"status":       model.Status,
		"id":           model.ID,
		"version":      model.Version,
		"user_id":      userAuth.GetID(),
	}

	parsedQuery, args := sqlformat.NamedQuery(query, params)
	r.logger.PrintInfo(sqlformat.MinifySQL(parsedQuery), nil)

	err := tx.QueryRowContext(ctx, parsedQuery, args...).Scan(&model.Version)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apiError.ErrEditConflict
		}
		return err
	}

	return nil
}

func (r *OrderRepository) DeleteById(
	ctx context.Context,
	id uuid.UUID,
) error {
	tx := contexts.GetTx(ctx)
	if tx == nil {
		panic("transaction necessary for this operation")
	}

	userAuth := contexts.GetUser(ctx)

	query := `
        UPDATE orders
        SET
            deleted = true,
            updated_at = NOW(),
            updated_by = :user_id,
            version = version + 1
        WHERE id = :id
            AND deleted = false
        RETURNING version
    `

	params := map[string]any{
		"id":      id,
		"user_id": userAuth.GetID(),
	}

	parsedQuery, args := sqlformat.NamedQuery(query, params)
	r.logger.PrintInfo(sqlformat.MinifySQL(parsedQuery), nil)

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return apiError.ErrRecordNotFound
	}

	queryItems := `
        UPDATE order_items
        SET
            deleted = true,
            updated_at = NOW(),
            updated_by = :user_id,
            version = version + 1
        WHERE order_id = :orderID
            AND deleted = false
    `

	paramsItems := map[string]any{
		"orderID": id,
		"user_id": userAuth.GetID(),
	}

	parsedItemsQuery, args := sqlformat.NamedQuery(queryItems, paramsItems)
	r.logger.PrintInfo(sqlformat.MinifySQL(parsedItemsQuery), nil)

	result, err = tx.ExecContext(ctx, parsedItemsQuery, args...)
	if err != nil {
		return err
	}

	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return apiError.ErrRecordNotFound
	}

	return nil
}
