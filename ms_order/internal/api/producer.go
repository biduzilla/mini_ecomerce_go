package api

import (
	"ms_order/internal/core/jsonlog"
	"ms_order/internal/core/messaging"

	"github.com/IBM/sarama"
)

type producers struct {
	orderProducer *messaging.OrderProducer
}

func NewProducers(
	producer sarama.SyncProducer,
	logger jsonlog.Logger,
) *producers {
	return &producers{
		orderProducer: messaging.NewOrderProducer(producer, logger),
	}
}
