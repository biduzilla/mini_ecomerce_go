package order

import (
	"database/sql/driver"
	"errors"
	"ms_order/internal/core/domain/models"
	"ms_order/internal/core/validator"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusApproved OrderStatus = "APPROVED"
	OrderStatusRejected OrderStatus = "REJECTED"
)

func IsValidOrderStatus(status OrderStatus) bool {
	switch status {
	case OrderStatusPending, OrderStatusApproved, OrderStatusRejected:
		return true
	default:
		return false
	}
}

type Order struct {
	models.BaseModel
	ID          uuid.UUID
	TotalAmount float64
	Status      OrderStatus
}

type OrderItem struct {
	models.BaseModel
	ID        uuid.UUID
	OrderID   uuid.UUID
	ProductID uuid.UUID
	Quantity  int
	UnitPrice float64
}

type OrderDTO struct {
	ID          *uuid.UUID     `json:"id"`
	Items       []OrderItemDTO `json:"items"`
	TotalAmount *float64       `json:"totalAmount"`
	Status      *OrderStatus   `json:"status"`
	CreatedAt   *time.Time     `json:"createdAt"`
}

type OrderItemDTO struct {
	ID        *uuid.UUID `json:"id"`
	ProductID *uuid.UUID `json:"productId"`
	Quantity  *int       `json:"quantity"`
	UnitPrice *float64   `json:"unitPrice"`
}

func (m *Order) ToDTO() *OrderDTO {
	return &OrderDTO{
		ID:          &m.ID,
		TotalAmount: &m.TotalAmount,
		Status:      &m.Status,
		CreatedAt:   &m.CreatedAt,
	}
}

func (m *OrderItem) ToDTO() *OrderItemDTO {
	return &OrderItemDTO{
		ID:        &m.ID,
		ProductID: &m.ProductID,
		Quantity:  &m.Quantity,
		UnitPrice: &m.UnitPrice,
	}
}

func (d OrderDTO) ToModel() *Order {
	var model Order

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.TotalAmount != nil {
		model.TotalAmount = *d.TotalAmount
	}

	if d.Status != nil {
		model.Status = *d.Status
	} else {
		model.Status = OrderStatusPending
	}

	return &model
}

func (d OrderItemDTO) ToModel(orderID uuid.UUID) *OrderItem {
	var model OrderItem

	if d.ID != nil {
		model.ID = *d.ID
	}

	model.OrderID = orderID

	if d.ProductID != nil {
		model.ProductID = *d.ProductID
	}

	if d.Quantity != nil {
		model.Quantity = *d.Quantity
	}

	if d.UnitPrice != nil {
		model.UnitPrice = *d.UnitPrice
	}

	return &model
}

func (o *Order) Validate(v *validator.Validator) {
	v.Check(o.TotalAmount >= 0, "totalAmount", "must be greater than or equal to 0")
	v.Check(IsValidOrderStatus(o.Status), "status", "must be a valid order status (PENDING, APPROVED, REJECTED)")
}

func (i *OrderItem) Validate(v *validator.Validator) {
	v.Check(i.OrderID != uuid.Nil, "orderId", "must be provided")
	v.Check(i.ProductID != uuid.Nil, "productId", "must be provided")
	v.Check(i.Quantity > 0, "quantity", "must be greater than 0")
	v.Check(i.UnitPrice > 0, "unitPrice", "must be greater than 0")
}

func ValidateOrderItems(v *validator.Validator, items []OrderItemDTO) {
	v.Check(len(items) > 0, "items", "must have at least one item")
}

func ValidateOrderDTO(v *validator.Validator, dto OrderDTO) {
	ValidateOrderItems(v, dto.Items)

	for i, item := range dto.Items {
		v.Check(item.ProductID != nil, "items["+string(rune(i))+"].productId", "must be provided")
		v.Check(item.Quantity != nil, "items["+string(rune(i))+"].quantity", "must be provided")
		v.Check(item.UnitPrice != nil, "items["+string(rune(i))+"].unitPrice", "must be provided")

		if item.Quantity != nil {
			v.Check(*item.Quantity > 0, "items["+string(rune(i))+"].quantity", "must be greater than 0")
		}
		if item.UnitPrice != nil {
			v.Check(*item.UnitPrice > 0, "items["+string(rune(i))+"].unitPrice", "must be greater than 0")
		}
	}
}

func (o *OrderStatus) Scan(value any) error {
	if value == nil {
		*o = ""
		return nil
	}

	switch v := value.(type) {
	case string:
		*o = OrderStatus(v)
	case []byte:
		*o = OrderStatus(v)
	default:
		return errors.New("invalid type for OrderStatus")
	}

	return nil
}

func (o OrderStatus) Value() (driver.Value, error) {
	return string(o), nil
}

func ItemsToModels(dtos []OrderItemDTO, orderID uuid.UUID) ([]OrderItem, error) {
	items := make([]OrderItem, len(dtos))

	for i, dto := range dtos {
		items[i] = *dto.ToModel(orderID)
	}

	return items, nil
}

func CalculateTotalFromItems(items []*OrderItem) float64 {
	var total float64
	for _, item := range items {
		total += float64(item.Quantity) * item.UnitPrice
	}
	return total
}
