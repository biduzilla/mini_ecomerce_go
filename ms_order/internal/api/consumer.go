package api

import (
	"context"
	"fmt"
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

func (c *consumers) Start(logger jsonlog.Logger) (shutdownFunc func()) {
	logger.PrintInfo("starting kafka stock consumer...", nil)

	consumerCtx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		if err := c.stockConsumers.Start(consumerCtx); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	shutdownFunc = func() {
		logger.PrintInfo("stopping kafka stock consumer...", nil)
		cancel()
		<-errCh
		logger.PrintInfo("kafka stock consumer stopped gracefully", nil)
	}

	go func() {
		if err := <-errCh; err != nil {
			logger.PrintError(fmt.Errorf("kafka consumer error: %w", err), nil)
		}
	}()

	return shutdownFunc
}
