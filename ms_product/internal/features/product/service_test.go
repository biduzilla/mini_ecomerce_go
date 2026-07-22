package product

import (
	"context"
	"errors"
	"ms_product/internal/core/domain/apiError"
	"ms_product/internal/core/domain/models"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockProductRepo struct {
	insertFn    func(ctx context.Context, model *Product) error
	insertAllFn func(
		ctx context.Context,
		models []*Product,
	) error
	getByIDFn func(ctx context.Context, id uuid.UUID) (*Product, error)
	updateFn  func(ctx context.Context, model *Product) error
	deleteFn  func(ctx context.Context, id uuid.UUID) error
}

func (m *mockProductRepo) Insert(ctx context.Context, model *Product) error {
	return m.insertFn(ctx, model)
}

func (m *mockProductRepo) InsertAll(
	ctx context.Context,
	models []*Product,
) error {
	return m.insertAllFn(ctx, models)
}

func (m *mockProductRepo) GetByID(ctx context.Context, id uuid.UUID) (*Product, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockProductRepo) Update(ctx context.Context, model *Product) error {
	return m.updateFn(ctx, model)
}

func (m *mockProductRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
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

func (m *mockKeyBuilder) BuildItemKey(id string) string     { return "product:" + id }
func (m *mockKeyBuilder) BuildListKey(params ...any) string { return "product:list" }
func (m *mockKeyBuilder) GetPrefix() string                 { return "product:" }

func newValidProduct() *Product {
	return &Product{
		ID:    uuid.New(),
		Name:  "name",
		Price: 5000.,
	}
}

func TestFindByID_NotFound(t *testing.T) {
	productExpected := newValidProduct()

	repo := &mockProductRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Product, error) {
			return nil, apiError.ErrRecordNotFound
		},
	}
	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error {
		return errors.New("cache miss")
	}}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{})

	_, err := svc.GetByID(context.Background(), productExpected.ID)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, apiError.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}

func TestFindByID_Sucess(t *testing.T) {
	expectedProduct := newValidProduct()

	repo := &mockProductRepo{
		getByIDFn: func(ctx context.Context, id uuid.UUID) (*Product, error) {
			return expectedProduct, nil
		},
	}
	cacheMock := &mockCache{getFn: func(ctx context.Context, key string, dest any) error {
		return errors.New("cache miss")
	}}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{})

	result, err := svc.GetByID(context.Background(), expectedProduct.ID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.ID != expectedProduct.ID {
		t.Error("Returned wrong product ID")
	}
}

func TestInsert_ValidationFailure_ShortCircuit(t *testing.T) {
	invalidModel := &Product{
		ID:    uuid.New(),
		Name:  "",
		Price: 0.,
	}
	insertCalled := false
	repo := &mockProductRepo{
		insertFn: func(ctx context.Context, model *Product) error {
			insertCalled = true
			return nil
		},
	}
	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{})

	err := svc.Create(context.Background(), invalidModel)
	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}
	var valErr *apiError.ValidationError
	if !errors.As(err, &valErr) {
		t.Errorf("Expected ValidationError, got %T", err)
	}
	if insertCalled {
		t.Error("Database Insert was called for invalid data! Should have short-circuited.")
	}

}

func TestCreate_DuplicateName_Conflict(t *testing.T) {
	product := newValidProduct()
	duplicateNameErr := apiError.ValidationAlreadyExists("name")

	repo := &mockProductRepo{
		insertFn: func(ctx context.Context, model *Product) error {
			return duplicateNameErr
		},
	}

	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{})

	err := svc.Create(context.Background(), product)

	if err == nil {
		t.Fatal("Expected duplicate name error, got nil")
	}
	if !errors.Is(err, duplicateNameErr) {
		t.Errorf("Expected duplicate name error, got %v", err)
	}
}

func TestCreate_Success(t *testing.T) {
	validProduct := &Product{
		Name:  "Mouse Logitech",
		Price: 150.00,
	}

	repo := &mockProductRepo{insertFn: func(ctx context.Context, model *Product) error { return nil }}

	cacheCleared := false
	cacheMock := &mockCache{
		deleteByPrefixFn: func(ctx context.Context, prefix string) error {
			cacheCleared = true
			return nil
		},
	}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{})

	err := svc.Create(context.Background(), validProduct)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if !cacheCleared {
		t.Error("Cache was not cleared after successful insert")
	}
}

func TestCreateAll_ValidationFailure(t *testing.T) {
	invalidProducts := []*Product{
		{Name: ""},
	}

	insertCalled := false
	repo := &mockProductRepo{
		insertAllFn: func(ctx context.Context, models []*Product) error {
			insertCalled = true
			return nil
		},
	}

	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{})

	err := svc.CreateAll(context.Background(), invalidProducts)

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}

	if insertCalled {
		t.Error("Database Insert was called for invalid data! Should have short-circuited.")
	}
}

func TestCreateAll_Success(t *testing.T) {
	validItems := []*Product{
		{Name: "Teclado", Price: 100.0},
		{Name: "Mouse", Price: 50.0},
	}

	insertAllCalled := false
	repo := &mockProductRepo{
		insertAllFn: func(ctx context.Context, models []*Product) error {
			insertAllCalled = true
			return nil
		},
	}
	cacheCleared := false
	cacheMock := &mockCache{
		deleteByPrefixFn: func(ctx context.Context, prefix string) error {
			cacheCleared = true
			return nil
		},
	}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{})

	err := svc.CreateAll(context.Background(), validItems)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !insertAllCalled {
		t.Error("InsertAll was not called for valid data")
	}

	time.Sleep(10 * time.Millisecond)

	if !cacheCleared {
		t.Error("Cache was not cleared after successful insert")
	}
}

func TestUpdate_ValidationFailure_ShortCircuit(t *testing.T) {
	invalidProduct := &Product{Name: ""}

	updateCalled := false
	repo := &mockProductRepo{
		updateFn: func(ctx context.Context, model *Product) error {
			updateCalled = true
			return nil
		},
	}

	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{})

	err := svc.Update(context.Background(), invalidProduct)

	if err == nil {
		t.Fatal("Expected validation error, got nil")
	}
	if updateCalled {
		t.Error("Database Update was called for invalid data!")
	}
}

func TestUpdate_Conflict_OrNotFound(t *testing.T) {
	validProduct := &Product{
		ID:        uuid.New(),
		Name:      "Mouse",
		Price:     50.0,
		BaseModel: models.BaseModel{Version: 1},
	}

	repo := &mockProductRepo{
		updateFn: func(ctx context.Context, model *Product) error {
			return apiError.ErrEditConflict
		},
	}

	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{})

	err := svc.Update(context.Background(), validProduct)

	if !errors.Is(err, apiError.ErrEditConflict) {
		t.Errorf("Expected ErrEditConflict (Optimistic Lock), got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := &mockProductRepo{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			return apiError.ErrRecordNotFound
		},
	}

	svc := NewService(repo, &mockTxManager{}, &mockCache{}, &mockKeyBuilder{})

	err := svc.Delete(context.Background(), uuid.New())

	if !errors.Is(err, apiError.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}

func TestUpdate_Success(t *testing.T) {
	validItem := newValidProduct()

	repoCalled := false
	repo := &mockProductRepo{
		updateFn: func(ctx context.Context, model *Product) error {
			repoCalled = true
			return nil
		},
	}
	cacheCleared := false
	cacheMock := &mockCache{
		deleteByPrefixFn: func(ctx context.Context, prefix string) error {
			cacheCleared = true
			return nil
		},
	}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{})

	err := svc.Update(context.Background(), validItem)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !repoCalled {
		t.Error("Repo was not called for valid data")
	}

	time.Sleep(10 * time.Millisecond)

	if !cacheCleared {
		t.Error("Cache was not cleared after successful insert")
	}
}

func TestDelete_Success(t *testing.T) {
	deleteCalled := false
	repo := &mockProductRepo{
		deleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	cacheCleared := false
	cacheMock := &mockCache{
		deleteByPrefixFn: func(ctx context.Context, prefix string) error {
			cacheCleared = true
			return nil
		},
	}

	svc := NewService(repo, &mockTxManager{}, cacheMock, &mockKeyBuilder{})

	err := svc.Delete(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !deleteCalled {
		t.Error("Database Delete was not called!")
	}

	time.Sleep(10 * time.Millisecond)

	if !cacheCleared {
		t.Error("Cache was not cleared!")
	}
}
