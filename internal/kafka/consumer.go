// Package kafka provides an adapter for consuming messages from a Kafka topic.
//
// It uses the sarama library to interact with Kafka and implements a consumer group pattern.
//
// The ConsumerAdapter listens for messages from the specified topic,
// processes them using a provided handler function, and manages Kafka consumer group sessions.
//
// The package supports graceful shutdown and error handling during message processing.
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

// ConsumerAdapter represents a Kafka consumer group that consumes messages from a specific topic.
// It wraps around the `sarama.ConsumerGroup` and holds the topic name to facilitate message consumption.
type ConsumerAdapter struct {
	consumerGroup sarama.ConsumerGroup
	topic         string
}

// NewConsumerAdapter creates a new instance of ConsumerAdapter,
// which is a Kafka consumer group that consumes messages from a specified topic.
//
// It sets up the necessary Kafka consumer configurations,
// including the version and rebalance strategy,
// and initializes the consumer group for the provided brokers.
func NewConsumerAdapter(brokerCfg config.Broker) (*ConsumerAdapter, error) {
	const fn = "NewConsumerAdapter"

	config := sarama.NewConfig()
	config.Version = sarama.V3_6_0_0 // Установите версию Kafka
	config.Consumer.Group.Rebalance.Strategy = sarama.NewBalanceStrategyRoundRobin()

	consumerGroup, err := sarama.NewConsumerGroup(brokerCfg.Hosts, brokerCfg.GroupID, config)
	if err != nil {
		return nil, fmt.Errorf("(%s) | Error creating ConsumerAdapter: %w", fn, err)
	}

	log.Printf("(%s) | Kafka created!\n", fn)

	return &ConsumerAdapter{
		consumerGroup: consumerGroup,
		topic:         brokerCfg.Topic,
	}, nil
}

// Start begins the consumption of messages from the Kafka topic.
// It continuously consumes messages using the provided consumer group
// and processes them using the provided message handler function.
//
// The consumer will keep running in a loop until the provided context is canceled or an error occurs.
func (k *ConsumerAdapter) Start(ctx context.Context, messageHandler func(order models.Order) error) {
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

// Close gracefully shuts down the Kafka consumer group, releasing any resources associated with it.
//
// This method ensures that the consumer group is closed and any ongoing consumption is properly terminated.
func (k *ConsumerAdapter) Close() error {
	return k.consumerGroup.Close()
}

// kafkaConsumerHandler represents the Kafka message handler for the consumer,
// which accepts messages from Kafka and passes them to the processing function.
type kafkaConsumerHandler struct {
	messageHandler func(order models.Order) error
}

// Setup - is called before the start of processing. Doesn't do anything yet.
func (h *kafkaConsumerHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup - is called after the processing is complete. Doesn't do anything yet.
func (h *kafkaConsumerHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a Kafka consumer group claim.
// It unmarshals each message's value into an `Order` object
// and processes it using the provided message handler.
// After processing each message, it marks the message as processed in the session.
// If there are errors in unmarshalling or processing,
// they are logged but do not interrupt the consumption of further messages.
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
