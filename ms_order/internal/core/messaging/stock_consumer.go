// internal/core/messaging/stock_consumer.go
package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"ms_order/internal/core/events"
	"ms_order/internal/features/order"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type StockEventConsumer struct {
	orderService orderService
}

type orderService interface {
	UpdateOrderStatus(ctx context.Context, id uuid.UUID, status order.OrderStatus) error
}

func NewStockEventConsumer(orderService orderService) *StockEventConsumer {
	return &StockEventConsumer{
		orderService: orderService,
	}
}

func (c *StockEventConsumer) handleAvailabilityCheck(event *events.AvailabilityCheckEvent) error {
	slog.Info("📩 Recebido stock.check-result", "eventId", event.EventID, "orderId", event.OrderID)

	var status order.OrderStatus
	if event.Available {
		slog.Info("Estoque suficiente para pedido", "orderId", event.OrderID)
		status = order.OrderStatusApproved
	} else {
		slog.Warn("Estoque insuficiente para pedido", "orderId", event.OrderID)
		status = order.OrderStatusRejected
	}

	err := c.orderService.UpdateOrderStatus(context.Background(), event.OrderID, status)
	if err != nil {
		slog.Error("Erro ao atualizar status do pedido", "orderId", event.OrderID, "error", err)
		return err
	}

	slog.Info("✅ Pedido atualizado para status", "orderId", event.OrderID, "status", status)
	return nil
}

var _ sarama.ConsumerGroupHandler = (*StockEventConsumer)(nil)

func (c *StockEventConsumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *StockEventConsumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *StockEventConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			var event events.AvailabilityCheckEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				slog.Error("Falha ao deserializar evento de estoque", "error", err)
				session.MarkMessage(msg, "")
				continue
			}

			if err := c.handleAvailabilityCheck(&event); err != nil {
				return err
			}

			session.MarkMessage(msg, "")
		}
	}
}
