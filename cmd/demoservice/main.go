package main

import (
	"context"
	"demo_service/internal/cache"
	"demo_service/internal/config"
	"demo_service/internal/db"
	"demo_service/internal/kafka"
	"demo_service/internal/models"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.MustLoad()
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	storage, err := db.NewPgsql(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("Fatal ERROR: %v", err)
	}

	cache := cache.NewCahce(cfg.Cache.Capacity)
	if orders, err := storage.GetLastLimitOrders(ctx, cfg.Cache.Capacity); err != nil {
		log.Fatalf("Error getting last orders: %v\n", err)
	} else if len(orders) == 0 {
		log.Print("DB is empty")
	} else {
		log.Println("The cache filling has started...")
		for _, order := range orders {
			if !cache.Set(order.OrderUID, order) {
				log.Printf("Error saving order: %v\n", order)
			}
		}
		log.Println("The cache has been filled!")
	}

	// Создаем адаптер Kafka
	kfkAdapter, err := kafka.NewKafkaAdapter(cfg.Broker)
	if err != nil {
		log.Fatalf("Error creating a Kafka adapter: %v\n", err)
	}
	defer kfkAdapter.Close()

	// Канал для обработки сигнала завершения
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	log.Print("Starting service...")
	go func() {
		kfkAdapter.Start(ctx, func(order models.Order) error {
			log.Printf("Message received: %s\n", order.OrderUID)
			if err := storage.SaveOrder(ctx, order); err != nil {
				log.Printf("Error saving order: %v\n", err)
				return err
			}
			if !cache.Set(order.OrderUID, order) {
				log.Printf("Error saving order: %v\n", order)
			}
			return nil
		})
	}()

	// Ожидание сигнала завершения
	<-signalChan
	// Завершение работы
	log.Println("Completion of work...")
	time.Sleep(2 * time.Second)
	storage.Close()
	ctxCancel()
}
