package api

import (
	"context"
	"fmt"
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

func (c *consumers) Start(logger jsonlog.Logger) (shutdownFunc func()) {
	logger.PrintInfo("starting kafka stock consumer...", nil)

	consumerCtx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		if err := c.order.Start(consumerCtx); err != nil {
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
