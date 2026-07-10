package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"ms_stock/internal/core/events"
	"ms_stock/internal/core/jsonlog"

	"github.com/IBM/sarama"
)

const StockTopic = "stock.check-result"

type StockEventProducer struct {
	producer sarama.SyncProducer
	logger   jsonlog.Logger
}

func NewStockEventProducer(
	producer sarama.SyncProducer,
	logger jsonlog.Logger,
) *StockEventProducer {
	return &StockEventProducer{
		producer: producer,
		logger:   logger,
	}
}

func (p *StockEventProducer) PublishAvailabilityCheck(
	ctx context.Context,
	event events.AvailabilityCheckEvent,
) error {
	value, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal order created event: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: StockTopic,
		Key:   sarama.StringEncoder(event.OrderID.String()),
		Value: sarama.ByteEncoder(value),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.logger.PrintError(err,
			map[string]string{
				"eventId": event.EventID.String(),
				"orderId": event.OrderID.String(),
				"message": "Failed to publish order created event",
			})

		return fmt.Errorf("failed to publish order created event: %w", err)
	}

	p.logger.PrintInfo("Order created event published",
		map[string]string{
			"eventId":   event.EventID.String(),
			"orderId":   event.OrderID.String(),
			"partition": string(partition),
			"offset":    fmt.Sprint(offset),
		})

	return nil
}
