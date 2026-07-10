package api

import (
	"ms_stock/internal/core/jsonlog"
	"ms_stock/internal/core/messaging"

	"github.com/IBM/sarama"
)

type producers struct {
	stock *messaging.StockEventProducer
}

func NewProducers(
	producer sarama.SyncProducer,
	logger jsonlog.Logger,
) *producers {
	return &producers{
		stock: messaging.NewStockEventProducer(producer, logger),
	}
}
