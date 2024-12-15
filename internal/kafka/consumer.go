package kafka

import (
	"context"
	"demo_service/internal/config"
	"demo_service/internal/models"
	"encoding/json"
	"fmt"
	"log"

	"github.com/IBM/sarama"
)

// KafkaAdapter - структура для работы с Kafka
type KafkaAdapter struct {
	consumerGroup sarama.ConsumerGroup
	topic         string
}

// NewKafkaAdapter - конструктор адаптера Kafka
func NewKafkaAdapter(broker_cfg config.Broker) (*KafkaAdapter, error) {
	const fn = "NewKafkaAdapter"

	config := sarama.NewConfig()
	config.Version = sarama.V3_6_0_0 // Установите версию Kafka
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()

	consumerGroup, err := sarama.NewConsumerGroup(broker_cfg.Hosts, broker_cfg.GroupID, config)
	if err != nil {
		return nil, fmt.Errorf("(%s) | Error creating kafkaAdapter: %w", fn, err)
	}

	log.Printf("(%s) | Kafka created!\n", fn)

	return &KafkaAdapter{
		consumerGroup: consumerGroup,
		topic:         broker_cfg.Topic,
	}, nil
}

// Start - запуск адаптера для обработки сообщений
func (k *KafkaAdapter) Start(ctx context.Context, messageHandler func(order models.Order) error) {
	const fn = "Start"

	handler := &kafkaConsumerHandler{
		messageHandler: messageHandler,
	}

	for {
		if err := k.consumerGroup.Consume(ctx, []string{k.topic}, handler); err != nil {
			log.Printf("(%s) | Error reading messages: %v\n", fn, err)
		}
		// Если контекст завершен, выходим из цикла
		if ctx.Err() != nil {
			break
		}
	}
}

// Close - завершение работы с Kafka
func (k *KafkaAdapter) Close() error {
	return k.consumerGroup.Close()
}

// kafkaConsumerHandler - обработчик сообщений Kafka
type kafkaConsumerHandler struct {
	messageHandler func(order models.Order) error
}

// Setup - вызывается перед началом обработки
func (h *kafkaConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup - вызывается после завершения обработки
func (h *kafkaConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim - обработка сообщений
func (h *kafkaConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	const fn = "ConsumeClaim"
	for message := range claim.Messages() {
		var order models.Order
		if err := json.Unmarshal(message.Value, &order); err != nil {
			log.Printf("(%s) | Error unmarshalling JSON: %v\n", fn, err)
			continue
		}

		if err := h.messageHandler(order); err != nil {
			log.Printf("(%s) | Message processing error: %v\n", fn, err)
		}

		session.MarkMessage(message, "")
	}
	return nil
}