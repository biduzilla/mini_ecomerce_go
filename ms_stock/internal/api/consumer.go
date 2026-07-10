package api

import (
	"ms_stock/internal/core/jsonlog"
	"ms_stock/internal/core/messaging"

	"github.com/IBM/sarama"
)

type consumers struct {
	order *messaging.OrderEventConsumer
}

func NewConsumer(
	consumerGroup sarama.ConsumerGroup,
	services *services,
	logger jsonlog.Logger,
	producers *producers,
) *consumers {
	return &consumers{
		order: messaging.NewOrderEventConsumer(
			consumerGroup,
			services.stockService,
			logger,
			producers.stock,
		),
	}
}
