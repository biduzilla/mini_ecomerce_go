package messaging

import (
	"context"
	"encoding/json"
	"ms_stock/internal/core/contexts"
	"ms_stock/internal/core/domain"
	"ms_stock/internal/core/events"
	"ms_stock/internal/core/jsonlog"
	"ms_stock/internal/features/stock"
	"time"

	"github.com/IBM/sarama"
)

type OrderEventConsumer struct {
	consumerGroup sarama.ConsumerGroup
	stockService  stockService
	logger        jsonlog.Logger
	stockProducer
}

type stockService interface {
	CheckAvailability(
		ctx context.Context,
		request stock.AvailabilityCheckRequest,
	) (*stock.AvailabilityCheckResponse, error)

	DeductStock(
		ctx context.Context,
		req stock.AvailabilityCheckRequest,
	) error
}

type stockProducer interface {
	PublishAvailabilityCheck(
		ctx context.Context,
		event events.AvailabilityCheckEvent,
	) error
}

func NewOrderEventConsumer(
	consumerGroup sarama.ConsumerGroup,
	stockService stockService,
	logger jsonlog.Logger,
	stockProducer stockProducer,
) *OrderEventConsumer {
	return &OrderEventConsumer{
		consumerGroup: consumerGroup,
		stockService:  stockService,
		logger:        logger,
		stockProducer: stockProducer,
	}
}

func (c *OrderEventConsumer) Start(ctx context.Context) error {
	topics := []string{"orders"}

	for {
		err := c.consumerGroup.Consume(ctx, topics, c)

		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			c.logger.PrintInfo("kafka topic not ready yet, retrying in 5s...", map[string]string{
				"error": err.Error(),
			})

			select {
			case <-ctx.Done():
				return nil
			case <-time.After(5 * time.Second):
				continue
			}
		}

		if ctx.Err() != nil {
			return nil
		}
	}
}

func (c *OrderEventConsumer) handleOrderCreated(event *events.OrderCreatedEvent) error {
	ctx := contexts.SetUser(context.Background(), domain.AnonymousUser)

	c.logger.PrintInfo("📩 Recebido OrderCreatedEvent",
		map[string]string{
			"eventId": event.EventID.String(),
			"orderId": event.OrderID.String(),
		},
	)

	itemsReq := make([]stock.ItemRequest, len(event.Items))
	for i, item := range event.Items {
		itemsReq[i] = stock.ItemRequest{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		}
	}
	checkReq := stock.AvailabilityCheckRequest{
		Items: itemsReq,
	}

	checkResponse, err := c.stockService.CheckAvailability(ctx, checkReq)
	if err != nil {
		c.logger.PrintError(err,
			map[string]string{
				"orderId": event.OrderID.String(),
				"message": "Erro ao processar pedido",
			},
		)
		return err
	}

	if checkResponse.Available {
		err = c.stockService.DeductStock(ctx, checkReq)
		if err != nil {
			c.logger.PrintError(err,
				map[string]string{
					"orderId": event.OrderID.String(),
					"message": "Erro ao processar pedido",
				},
			)
			return err
		}
	}

	detailsEvent := make([]events.ItemAvailabilityDetailEvent, len(checkResponse.Details))
	for i, d := range checkResponse.Details {
		detailsEvent[i] = events.ItemAvailabilityDetailEvent{
			ProductID: d.ProductID,
			Requested: int64(d.Requested),
			Available: int64(d.Available),
		}
	}

	availabilityEvent := events.AvailabilityCheckEvent{
		EventID:   event.EventID,
		Timestamp: event.Timestamp,
		OrderID:   event.OrderID,
		Available: checkResponse.Available,
		Details:   detailsEvent,
	}

	err = c.stockProducer.PublishAvailabilityCheck(ctx, availabilityEvent)
	if err != nil {
		c.logger.PrintError(err,
			map[string]string{
				"orderId": event.OrderID.String(),
				"message": err.Error(),
			},
		)
	}

	available := "false"
	if checkResponse.Available {
		available = "true"
	}
	c.logger.PrintInfo("✅ Verificação de estoque para pedido",
		map[string]string{
			"OrderID":    event.OrderID.String(),
			"disponível": available,
		},
	)

	return nil
}

var _ sarama.ConsumerGroupHandler = (*OrderEventConsumer)(nil)

func (c *OrderEventConsumer) Setup(sarama.ConsumerGroupSession) error {
	c.logger.PrintInfo("✅ Kafka consumer connected successfully, waiting for messages...", nil)
	return nil
}
func (c *OrderEventConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	c.logger.PrintInfo("Kafka consumer disconnected, releasing resources...", nil)
	return nil
}

func (c *OrderEventConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			var event events.OrderCreatedEvent
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				c.logger.PrintError(err, map[string]string{
					"message": "Falha ao deserializar evento de stock",
				})
				session.MarkMessage(msg, "")
				continue
			}

			if err := c.handleOrderCreated(&event); err != nil {
				return err
			}

			session.MarkMessage(msg, "")
		}
	}
}
