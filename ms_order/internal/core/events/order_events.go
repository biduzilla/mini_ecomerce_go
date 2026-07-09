package events

import (
	"time"

	"github.com/google/uuid"
)

type OrderItemEvent struct {
	ID         uuid.UUID `json:"id"`
	ProductID  uuid.UUID `json:"productId"`
	Quantity   int       `json:"quantity"`
	UnitPrice  float64   `json:"unitPrice"`
	TotalPrice float64   `json:"totalPrice"`
}

type OrderCreatedEvent struct {
	EventID     uuid.UUID        `json:"eventId"`
	OrderID     uuid.UUID        `json:"orderId"`
	Timestamp   time.Time        `json:"timestamp"`
	Status      string           `json:"status"`
	TotalAmount float64          `json:"totalAmount"`
	Items       []OrderItemEvent `json:"items"`
}

func NewOrderCreatedEvent(orderID uuid.UUID, status string, totalAmount float64, items []OrderItemEvent) *OrderCreatedEvent {
	return &OrderCreatedEvent{
		EventID:     uuid.New(),
		OrderID:     orderID,
		Timestamp:   time.Now().UTC(),
		Status:      status,
		TotalAmount: totalAmount,
		Items:       items,
	}
}
