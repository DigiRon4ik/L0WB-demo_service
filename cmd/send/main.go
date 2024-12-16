// Package main contains the main entry point for the Kafka producer application.
package main

import (
	"demo_service/internal/config"
	"demo_service/internal/models"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/IBM/sarama"
)

func main() {
	// Загружаем конфигурацию брокера
	cfg := config.MustLoad()
	// fmt.Println(cfg)

	// Создаем новый продюсер Kafka
	producer, err := newKafkaProducer(cfg.Broker)
	if err != nil {
		log.Fatalf("Ошибка при создании Kafka продюсера: %v\n", err)
	}
	defer producer.Close()

	// Пример отправки сообщения
	data, err := os.ReadFile("model.json")
	if err != nil {
		log.Fatalf("Ошибка чтения файла: %v", err)
	}

	var order models.Order
	err = json.Unmarshal(data, &order)
	if err != nil {
		log.Fatalf("Ошибка анмаршаллинга JSON: %v", err)
	}

	if err := sendMessage(producer, cfg.Broker.Topic, order); err != nil {
		log.Fatalf("Ошибка при отправке сообщения: %v\n", err)
	}
	log.Printf("Сообщение c Order_UID:(%s) отправлено в Kafka.", order.OrderUID)
}

// newKafkaProducer - создает новый Kafka продюсер
func newKafkaProducer(brokerCfg config.Broker) (sarama.SyncProducer, error) {
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // Ожидаем подтверждения от всех реплик
	config.Producer.Return.Successes = true          // Возвращаем успешные сообщения

	producer, err := sarama.NewSyncProducer(brokerCfg.Hosts, config)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании продюсера Kafka: %w", err)
	}

	return producer, nil
}

// sendMessage - отправка сообщения в Kafka
func sendMessage(producer sarama.SyncProducer, topic string, order models.Order) error {
	message, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("ошибка при маршаллинге сообщения: %w", err)
	}

	// Создаем сообщение для Kafka
	kafkaMessage := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(message),
	}

	// Отправляем сообщение в Kafka
	_, _, err = producer.SendMessage(kafkaMessage)
	if err != nil {
		return fmt.Errorf("ошибка при отправке сообщения в Kafka: %w", err)
	}

	return nil
}
