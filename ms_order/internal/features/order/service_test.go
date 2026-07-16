package order

import (
	"context"
	"errors"
	"ms_order/internal/core/clients/stock"
	"ms_order/internal/core/domain/apiError"
	"ms_order/internal/core/events"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockRepo struct {
	insertWithItemsFn    func(ctx context.Context, order *Order, items []*OrderItem) error
	findByIdFn           func(ctx context.Context, id uuid.UUID) (*Order, error)
	findByIdWithItemsFn  func(ctx context.Context, id uuid.UUID) (*Order, []*OrderItem, error)
	findItemsByOrderIdFn func(ctx context.Context, orderId uuid.UUID) ([]*OrderItem, error)
	updateFn             func(ctx context.Context, model *Order) error
	deleteByIdFn         func(ctx context.Context, id uuid.UUID) error
}

func (m *mockRepo) InsertWithItems(ctx context.Context, order *Order, items []*OrderItem) error {
	return m.insertWithItemsFn(ctx, order, items)
}
func (m *mockRepo) FindById(ctx context.Context, id uuid.UUID) (*Order, error) {
	return m.findByIdFn(ctx, id)
}
func (m *mockRepo) FindByIdWithItems(ctx context.Context, id uuid.UUID) (*Order, []*OrderItem, error) {
	return m.findByIdWithItemsFn(ctx, id)
}
func (m *mockRepo) FindItemsByOrderId(ctx context.Context, orderId uuid.UUID) ([]*OrderItem, error) {
	return m.findItemsByOrderIdFn(ctx, orderId)
}
func (m *mockRepo) Update(ctx context.Context, model *Order) error {
	return m.updateFn(ctx, model)
}
func (m *mockRepo) DeleteById(ctx context.Context, id uuid.UUID) error {
	return m.deleteByIdFn(ctx, id)
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

func (m *mockKeyBuilder) BuildItemKey(id string) string     { return "key:" + id }
func (m *mockKeyBuilder) BuildListKey(params ...any) string { return "key:list" }
func (m *mockKeyBuilder) GetPrefix() string                 { return "key:" }

type mockStockClient struct {
	response *stock.AvailabilityCheckResponse
	err      error
}

func (m *mockStockClient) CheckAvailability(ctx context.Context, req stock.AvailabilityCheckRequest) (*stock.AvailabilityCheckResponse, error) {
	return m.response, m.err
}

type mockProducer struct {
	err error
}

func (m *mockProducer) PublishOrderCreated(ctx context.Context, event *events.OrderCreatedEvent) error {
	return m.err
}

type mockLogger struct{}

func (m *mockLogger) PrintInfo(message string, properties map[string]string) {}
func (m *mockLogger) PrintError(err error, properties map[string]string)     {}
func (m *mockLogger) PrintFatal(err error, properties map[string]string)     {}
func (m *mockLogger) Write(message []byte) (n int, err error)                { return len(message), nil }

func newValidOrder() *Order {
	return &Order{
		ID:          uuid.New(),
		TotalAmount: 100.0,
		Status:      OrderStatusPending,
	}
}

func newValidItems(orderID uuid.UUID) []*OrderItem {
	return []*OrderItem{
		{
			ID:        uuid.New(),
			OrderID:   orderID,
			ProductID: uuid.New(),
			Quantity:  2,
			UnitPrice: 50.0,
		},
	}
}

func TestBuildOrderCreatedEvent(t *testing.T) {
	svc := &OrderService{}
	order := newValidOrder()
	items := newValidItems(order.ID)

	event := svc.buildOrderCreatedEvent(order, items)

	if event.OrderID != order.ID {
		t.Errorf("Expected OrderID %s, got %s", order.ID, event.OrderID)
	}
	if len(event.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", len(event.Items))
	}
	expectedTotalPrice := items[0].UnitPrice * float64(items[0].Quantity)
	if event.Items[0].TotalPrice != expectedTotalPrice {
		t.Errorf("Expected Item TotalPrice %f, got %f", expectedTotalPrice, event.Items[0].TotalPrice)
	}
}

func TestCreate_Success(t *testing.T) {
	repo := &mockRepo{insertWithItemsFn: func(ctx context.Context, order *Order, items []*OrderItem) error { return nil }}
	tx := &mockTxManager{}
	cacheMock := &mockCache{}
	kb := &mockKeyBuilder{}
	stockMock := &mockStockClient{}
	producerMock := &mockProducer{}
	logger := &mockLogger{}

	svc := NewService(repo, tx, cacheMock, kb, stockMock, producerMock, logger)
	order := newValidOrder()
	items := newValidItems(order.ID)

	err := svc.Create(context.Background(), order, items)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	expectedTotal := items[0].UnitPrice * float64(items[0].Quantity)
	if order.TotalAmount != expectedTotal {
		t.Errorf("Expected TotalAmount %f, got %f", expectedTotal, order.TotalAmount)
	}
}

func TestProcessOrder_FullFlow_Success(t *testing.T) {
	repo := &mockRepo{insertWithItemsFn: func(ctx context.Context, order *Order, items []*OrderItem) error { return nil }}
	tx := &mockTxManager{}
	cacheMock := &mockCache{}
	kb := &mockKeyBuilder{}
	stockMock := &mockStockClient{response: &stock.AvailabilityCheckResponse{Available: true}}
	producerMock := &mockProducer{}
	logger := &mockLogger{}

	svc := NewService(repo, tx, cacheMock, kb, stockMock, producerMock, logger)
	err := svc.processOrder(context.Background(), newValidOrder(), newValidItems(uuid.New()))
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestProcessOrder_StockUnavailable_AbortsEarly(t *testing.T) {
	insertCalled := false
	repo := &mockRepo{insertWithItemsFn: func(ctx context.Context, order *Order, items []*OrderItem) error {
		insertCalled = true
		return nil
	}}
	tx := &mockTxManager{}
	cacheMock := &mockCache{}
	kb := &mockKeyBuilder{}
	stockMock := &mockStockClient{response: &stock.AvailabilityCheckResponse{Available: false}}
	producerMock := &mockProducer{}
	logger := &mockLogger{}

	svc := NewService(repo, tx, cacheMock, kb, stockMock, producerMock, logger)
	err := svc.processOrder(context.Background(), newValidOrder(), newValidItems(uuid.New()))

	if err == nil {
		t.Fatal("Expected error due to stock, got nil")
	}
	if insertCalled {
		t.Error("InsertWithItems was called, but it should have aborted at stock check")
	}
}

func TestFindByID_Success_FromDB(t *testing.T) {
	order := newValidOrder()
	items := newValidItems(order.ID)

	repo := &mockRepo{
		findByIdWithItemsFn: func(ctx context.Context, id uuid.UUID) (*Order, []*OrderItem, error) {
			return order, items, nil
		},
	}
	tx := &mockTxManager{}

	cacheMock := &mockCache{
		getFn: func(ctx context.Context, key string, dest any) error {
			return errors.New("cache miss for testing")
		},
	}
	kb := &mockKeyBuilder{}
	stockMock := &mockStockClient{}
	producerMock := &mockProducer{}
	logger := &mockLogger{}

	svc := NewService(repo, tx, cacheMock, kb, stockMock, producerMock, logger)

	returnedOrder, returnedItems, err := svc.FindByID(context.Background(), order.ID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if returnedOrder.ID != order.ID {
		t.Errorf("Expected order ID %s, got %s", order.ID, returnedOrder.ID)
	}
	if len(returnedItems) != len(items) {
		t.Errorf("Expected %d items, got %d", len(items), len(returnedItems))
	}
}

func TestFindByID_NotFound(t *testing.T) {
	repo := &mockRepo{
		findByIdWithItemsFn: func(ctx context.Context, id uuid.UUID) (*Order, []*OrderItem, error) {
			return nil, nil, apiError.ErrRecordNotFound
		},
	}
	tx := &mockTxManager{}
	cacheMock := &mockCache{
		getFn: func(ctx context.Context, key string, dest any) error {
			return errors.New("cache miss")
		},
	}
	kb := &mockKeyBuilder{}
	stockMock := &mockStockClient{}
	producerMock := &mockProducer{}
	logger := &mockLogger{}

	svc := NewService(repo, tx, cacheMock, kb, stockMock, producerMock, logger)

	_, _, err := svc.FindByID(context.Background(), uuid.New())

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, apiError.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}

func TestUpdateStatus_Success(t *testing.T) {
	order := newValidOrder()

	repo := &mockRepo{
		findByIdWithItemsFn: func(ctx context.Context, id uuid.UUID) (*Order, []*OrderItem, error) {
			return order, []*OrderItem{}, nil
		},
		updateFn: func(ctx context.Context, model *Order) error {
			if model.Status != OrderStatusApproved {
				t.Errorf("Expected status to be updated to APPROVED before calling repo, got %s", model.Status)
			}
			return nil
		},
	}
	tx := &mockTxManager{}
	cacheMock := &mockCache{
		getFn: func(ctx context.Context, key string, dest any) error {
			return errors.New("cache miss")
		},
	}
	kb := &mockKeyBuilder{}
	stockMock := &mockStockClient{}
	producerMock := &mockProducer{}
	logger := &mockLogger{}

	svc := NewService(repo, tx, cacheMock, kb, stockMock, producerMock, logger)

	err := svc.UpdateStatus(context.Background(), order.ID, OrderStatusApproved)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestUpdateStatus_OrderNotFound(t *testing.T) {
	repo := &mockRepo{
		findByIdWithItemsFn: func(ctx context.Context, id uuid.UUID) (*Order, []*OrderItem, error) {
			return nil, nil, apiError.ErrRecordNotFound
		},
	}
	tx := &mockTxManager{}
	cacheMock := &mockCache{
		getFn: func(ctx context.Context, key string, dest any) error {
			return errors.New("cache miss")
		},
	}
	kb := &mockKeyBuilder{}
	stockMock := &mockStockClient{}
	producerMock := &mockProducer{}
	logger := &mockLogger{}

	svc := NewService(repo, tx, cacheMock, kb, stockMock, producerMock, logger)

	err := svc.UpdateStatus(context.Background(), uuid.New(), OrderStatusApproved)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}
	if !errors.Is(err, apiError.ErrRecordNotFound) {
		t.Errorf("Expected ErrRecordNotFound, got %v", err)
	}
}
