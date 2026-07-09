package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"ms_order/internal/core/events"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

const OrderTopic = "orders"

type OrderProducer struct {
	producer sarama.SyncProducer
	logger   *zap.Logger
}

func NewOrderProducer(
	producer sarama.SyncProducer,
	logger *zap.Logger,
) *OrderProducer {
	return &OrderProducer{
		producer: producer,
		logger:   logger,
	}
}

func (p *OrderProducer) PublishOrderCreated(ctx context.Context, event *events.OrderCreatedEvent) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order created event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: OrderTopic,
		Key:   sarama.StringEncoder(event.OrderID.String()),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.Error("Failed to publish order created event",
			zap.String("eventId", event.EventID.String()),
			zap.String("orderId", event.OrderID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to publish order created event: %w", err)
	}

	p.logger.Info("Order created event published",
		zap.String("eventId", event.EventID.String()),
		zap.String("orderId", event.OrderID.String()),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
	)

	return nil
}
