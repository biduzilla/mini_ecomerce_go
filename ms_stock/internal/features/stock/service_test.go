package stock

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"ms_stock/internal/core/clients/product"
	"ms_stock/internal/core/domain/apiError"

	"github.com/google/uuid"
)

// ==========================================
// 1. MOCKS
// ==========================================

type mockStockRepo struct {
	findByIdFn           func(ctx context.Context, id uuid.UUID) (*Stock, error)
	insertFn             func(ctx context.Context, model *Stock) error
	insertAllFn          func(ctx context.Context, models []*Stock) error
	updateFn             func(ctx context.Context, model *Stock) error
	deleteByIdFn         func(ctx context.Context, id uuid.UUID) error
	findAllByProductIdFn func(ctx context.Context, ids []uuid.UUID) ([]*Stock, error)
}

func (m *mockStockRepo) FindById(ctx context.Context, id uuid.UUID) (*Stock, error) {
	return m.findByIdFn(ctx, id)
}
func (m *mockStockRepo) Insert(ctx context.Context, model *Stock) error {
	return m.insertFn(ctx, model)
}
func (m *mockStockRepo) InsertAll(ctx context.Context, models []*Stock) error {
	return m.insertAllFn(ctx, models)
}
func (m *mockStockRepo) Update(ctx context.Context, model *Stock) error {
	return m.updateFn(ctx, model)
}
func (m *mockStockRepo) DeleteById(ctx context.Context, id uuid.UUID) error {
	return m.deleteByIdFn(ctx, id)
}
func (m *mockStockRepo) FindAllByProductIdIn(ctx context.Context, ids []uuid.UUID) ([]*Stock, error) {
	return m.findAllByProductIdFn(ctx, ids)
}

type mockTxManager struct {
	err error
}

func (m *mockTxManager) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.err != nil {
		return m.err
	}
	return fn(ctx)
}

type mockCache struct {
	getFn            func(ctx context.Context, key string, dest any) error
	setFn            func(ctx context.Context, key string, value any, ttl *time.Duration) error
	deleteFn         func(ctx context.Context, keys ...string) error
	deleteByPrefixFn func(ctx context.Context, prefix string) error
}

func (m *mockCache) Get(ctx context.Context, key string, dest any) error {
	if m.getFn != nil {
		return m.getFn(ctx, key, dest)
	}
	return nil
}
func (m *mockCache) Set(ctx context.Context, key string, value any, ttl *time.Duration) error {
	if m.setFn != nil {
		return m.setFn(ctx, key, value, ttl)
	}
	return nil
}
func (m *mockCache) Delete(ctx context.Context, keys ...string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, keys...)
	}
	return nil
}
func (m *mockCache) DeleteByPrefix(ctx context.Context, prefix string) error {
	if m.deleteByPrefixFn != nil {
		return m.deleteByPrefixFn(ctx, prefix)
	}
	return nil
}

type mockKeyBuilder struct{}

func (m *mockKeyBuilder) BuildItemKey(id string) string     { return "stock:" + id }
func (m *mockKeyBuilder) BuildListKey(params ...any) string { return "stock:list" }
func (m *mockKeyBuilder) GetPrefix() string                 { return "stock:" }

type mockProductClient struct {
	getByIdFn func(ctx context.Context, id uuid.UUID) (*product.ProductDTO, error)
}

func (m *mockProductClient) GetByID(ctx context.Context, id uuid.UUID) (*product.ProductDTO, error) {
	return m.getByIdFn(ctx, id)
}

// ==========================================
// 2. HELPERS
// ==========================================

func newValidStock(productID uuid.UUID, qty int) *Stock {
	return &Stock{
		ID:                uuid.New(),
		ProductId:         productID,
		AvailableQuantity: qty,
	}
}

func TestCreateStock_Success(t *testing.T) {
	productID := uuid.New()
	repo := &mockStockRepo{insertFn: func(ctx context.Context, model *Stock) error {
		return nil
	}}

	productMock := &mockProductClient{
		getByIdFn: func(ctx context.Context, id uuid.UUID) (*product.ProductDTO, error) {
			return &product.ProductDTO{}, nil
		},
	}

	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{}, productMock)
	stock := newValidStock(productID, 100)

	err := svc.CreateStock(context.Background(), stock)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestCreateStock_ProductNotFound_AbortsEarly(t *testing.T) {
	insertCalled := false
	repo := &mockStockRepo{insertFn: func(ctx context.Context, model *Stock) error {
		insertCalled = true
		return nil
	}}

	productMock := &mockProductClient{
		getByIdFn: func(ctx context.Context, id uuid.UUID) (*product.ProductDTO, error) {
			return nil, apiError.ErrRecordNotFound
		},
	}

	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{}, productMock)
	stock := newValidStock(uuid.New(), 100)

	err := svc.CreateStock(context.Background(), stock)
	if err == nil {
		t.Fatal("Expected error because product doesn't exist, got nil")
	}
	if insertCalled {
		t.Error("Insert was called! Stock was saved for a non-existent product.")
	}
}

func TestCheckAvailability_Available(t *testing.T) {
	p1 := uuid.New()
	p2 := uuid.New()

	repo := &mockStockRepo{
		findAllByProductIdFn: func(ctx context.Context, ids []uuid.UUID) ([]*Stock, error) {
			return []*Stock{
				newValidStock(p1, 10),
				newValidStock(p2, 5),
			}, nil
		},
	}

	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error {
		return errors.New("miss")
	}}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{}, &mockProductClient{})

	req := AvailabilityCheckRequest{
		Items: []ItemRequest{
			{ProductID: p1, Quantity: 5},
			{ProductID: p2, Quantity: 5},
		},
	}

	resp, err := svc.CheckAvailability(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !resp.Available {
		t.Error("Expected response.Available to be true")
	}
	if len(resp.Details) != 0 {
		t.Errorf("Expected 0 details of failure, got %d", len(resp.Details))
	}
}

func TestCheckAvailability_Insufficient(t *testing.T) {
	p1 := uuid.New()

	repo := &mockStockRepo{
		findAllByProductIdFn: func(ctx context.Context, ids []uuid.UUID) ([]*Stock, error) {
			return []*Stock{
				newValidStock(p1, 5),
			}, nil
		},
	}
	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error {
		return errors.New("miss")
	}}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{}, &mockProductClient{})
	req := AvailabilityCheckRequest{
		Items: []ItemRequest{
			{ProductID: p1, Quantity: 10},
		},
	}
	resp, err := svc.CheckAvailability(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp.Available {
		t.Error("Expected response.Available to be false")
	}
	if len(resp.Details) != 1 {
		t.Fatalf("Expected 1 detail of failure, got %d", len(resp.Details))
	}

	detail := resp.Details[0]
	if detail.Requested != 10 || detail.Available != 5 {
		t.Errorf("Detail report is wrong. Got Req: %d, Avail: %d", detail.Requested, detail.Available)
	}
}

func TestDeductStock_ProductNotFoundInMap(t *testing.T) {
	p1 := uuid.New()

	repo := &mockStockRepo{findAllByProductIdFn: func(ctx context.Context, ids []uuid.UUID) ([]*Stock, error) {
		return []*Stock{}, nil
	}}
	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error {
		return errors.New("miss")
	}}
	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{}, &mockProductClient{})
	req := AvailabilityCheckRequest{
		Items: []ItemRequest{
			{ProductID: p1, Quantity: 1},
		},
	}

	err := svc.DeductStock(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var apiErr *apiError.ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected ApiError, got %T", err)
	}

	if apiErr.Code != http.StatusConflict {
		t.Errorf("Expected 409 Conflict, got %d", apiErr.Code)
	}
}

func TestDeductStock_InsufficientQuantity(t *testing.T) {
	p1 := uuid.New()

	repo := &mockStockRepo{
		findAllByProductIdFn: func(ctx context.Context, ids []uuid.UUID) ([]*Stock, error) {
			return []*Stock{newValidStock(p1, 5)}, nil
		},
		updateFn: func(ctx context.Context, model *Stock) error {
			t.Error("Update was called! Should have aborted before updating DB")
			return nil
		},
	}
	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error {
		return errors.New("miss")
	}}
	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{}, &mockProductClient{})
	req := AvailabilityCheckRequest{
		Items: []ItemRequest{
			{ProductID: p1, Quantity: 10},
		},
	}
	err := svc.DeductStock(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	var apiErr *apiError.ApiError
	if !errors.As(err, &apiErr) {
		t.Fatalf("Expected ApiError, got %T", err)
	}
	if apiErr.Code != http.StatusConflict {
		t.Errorf("Expected 409, got %d", apiErr.Code)
	}
}

func TestDeductStock_Success(t *testing.T) {
	p1 := uuid.New()
	var updatedStock *Stock

	repo := &mockStockRepo{
		findAllByProductIdFn: func(ctx context.Context, ids []uuid.UUID) ([]*Stock, error) {
			return []*Stock{newValidStock(p1, 10)}, nil
		},
		updateFn: func(ctx context.Context, model *Stock) error {
			updatedStock = model
			return nil
		},
	}
	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error { return errors.New("miss") }}
	productMock := &mockProductClient{}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{}, productMock)

	req := AvailabilityCheckRequest{
		Items: []ItemRequest{{ProductID: p1, Quantity: 4}},
	}

	err := svc.DeductStock(context.Background(), req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if updatedStock == nil {
		t.Fatal("Update was not called")
	}

	if updatedStock.AvailableQuantity != 6 {
		t.Errorf("Expected available quantity to be 6, got %d", updatedStock.AvailableQuantity)
	}
}

func TestFindByID_Success_FromDB(t *testing.T) {
	expectedStock := newValidStock(uuid.New(), 50)

	repo := &mockStockRepo{findByIdFn: func(ctx context.Context, id uuid.UUID) (*Stock, error) {
		return expectedStock, nil
	}}
	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error {
		return errors.New("cache miss")
	}}
	productMock := &mockProductClient{}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{}, productMock)

	stock, err := svc.FindByID(context.Background(), expectedStock.ID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if stock.ID != expectedStock.ID {
		t.Error("Returned wrong stock")
	}
}
