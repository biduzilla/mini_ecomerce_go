// internal/api/consumers.go
package api

import (
	"ms_order/internal/core/jsonlog"
	"ms_order/internal/core/messaging"

	"github.com/IBM/sarama"
)

type consumers struct {
	stockConsumers *messaging.StockEventConsumer
}

func NewConsumers(
	kafkaConsumer sarama.ConsumerGroup,
	services *services,
	logger jsonlog.Logger,
) *consumers {
	return &consumers{
		stockConsumers: messaging.NewStockEventConsumer(kafkaConsumer, services.orderService, logger),
	}
}
