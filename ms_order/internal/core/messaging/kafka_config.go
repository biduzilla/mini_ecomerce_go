// internal/core/messaging/kafka.go (ou onde estiver seu InitKafka)
package messaging

import (
	"fmt"

	"github.com/IBM/sarama"
)

func InitKafka(brokers []string, groupID string) (sarama.SyncProducer, sarama.ConsumerGroup, error) {
	saramaCfg := sarama.NewConfig()
	saramaCfg.Producer.RequiredAcks = sarama.WaitForAll
	saramaCfg.Producer.Idempotent = true

	saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaCfg.Consumer.Offsets.AutoCommit.Enable = false

	saramaCfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}

	producer, err := sarama.NewSyncProducer(brokers, saramaCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start kafka producer: %w", err)
	}

	consumer, err := sarama.NewConsumerGroup(brokers, groupID, saramaCfg)
	if err != nil {
		producer.Close()
		return nil, nil, fmt.Errorf("failed to start kafka consumer: %w", err)
	}

	return producer, consumer, nil
}
