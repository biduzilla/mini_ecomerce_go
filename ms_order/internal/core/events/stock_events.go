package events

import (
	"time"

	"github.com/google/uuid"
)

type ItemAvailabilityDetailEvent struct {
	ProductID uuid.UUID `json:"productId"`
	Requested int64     `json:"requested"`
	Available int64     `json:"available"`
}

type AvailabilityCheckEvent struct {
	EventID   uuid.UUID                     `json:"eventId"`
	Timestamp time.Time                     `json:"timestamp"`
	OrderID   uuid.UUID                     `json:"orderId"`
	Available bool                          `json:"available"`
	Details   []ItemAvailabilityDetailEvent `json:"details"`
}
