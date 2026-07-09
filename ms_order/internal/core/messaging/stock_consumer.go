// internal/core/messaging/stock_consumer.go
package messaging

import (
	"context"
	"encoding/json"
	"log/slog"
	"ms_order/internal/core/events"
	"ms_order/internal/core/jsonlog"
	"ms_order/internal/features/order"

	"github.com/IBM/sarama"
	"github.com/google/uuid"
)

type StockEventConsumer struct {
	consumerGroup sarama.ConsumerGroup
	orderService  orderService
	logger        jsonlog.Logger
}

type orderService interface {
	UpdateStatus(ctx context.Context, id uuid.UUID, status order.OrderStatus) error
}

func NewStockEventConsumer(
	consumerGroup sarama.ConsumerGroup,
	orderService orderService,
	logger jsonlog.Logger,
) *StockEventConsumer {
	return &StockEventConsumer{
		consumerGroup: consumerGroup,
		orderService:  orderService,
		logger:        logger,
	}
}

func (c *StockEventConsumer) Start(ctx context.Context) error {
	topics := []string{"stock.check-result"}

	for {
		err := c.consumerGroup.Consume(ctx, topics, c)
		if err != nil {
			return err
		}

		if ctx.Err() != nil {
			return nil
		}
	}
}

func (c *StockEventConsumer) handleAvailabilityCheck(event *events.AvailabilityCheckEvent) error {
	c.logger.PrintInfo("📩 Recebido stock.check-result",
		map[string]string{
			"eventId": event.EventID.String(),
			"orderId": event.OrderID.String(),
		},
	)

	var status order.OrderStatus
	if event.Available {
		c.logger.PrintInfo("Estoque suficiente para pedido",
			map[string]string{
				"orderId": event.OrderID.String(),
			},
		)
		status = order.OrderStatusApproved
	} else {
		c.logger.PrintInfo("Estoque insuficiente para pedido",
			map[string]string{
				"orderId": event.OrderID.String(),
			},
		)
		status = order.OrderStatusRejected
	}

	err := c.orderService.UpdateStatus(context.Background(), event.OrderID, status)
	if err != nil {
		c.logger.PrintError(err,
			map[string]string{
				"orderId": event.OrderID.String(),
				"message": "Erro ao atualizar status do pedido",
			},
		)
		return err
	}

	c.logger.PrintInfo("✅ Pedido atualizado para status",
		map[string]string{
			"orderId": event.OrderID.String(),
			"status":  string(status),
		},
	)
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
