// Package main the original file.
package main

import (
	"context"
	"demo_service/internal/cache"
	"demo_service/internal/config"
	"demo_service/internal/db"
	"demo_service/internal/kafka"
	"demo_service/internal/models"
	orderModule "demo_service/internal/modules"
	"demo_service/internal/server"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	storage       *db.Storage
	cacheInstance *cache.Cache
	kfkAdapter    *kafka.ConsumerAdapter
)

func main() {
	var err error

	cfg := config.MustLoad()
	ctx, ctxCancel := context.WithCancel(context.Background())

	storage, err = db.New(ctx, cfg.DB)
	if err != nil {
		log.Fatalf("Fatal ERROR: %v", err)
	}

	cacheInstance = cache.New(cfg.Cache.Capacity)
	if err := cacheFill(ctx, cfg.Cache.Capacity); err != nil {
		log.Fatalf("Fatal ERROR: %v", err)
	}

	// Создаем адаптер Kafka
	kfkAdapter, err = kafka.NewConsumerAdapter(cfg.Broker)
	if err != nil {
		log.Fatalf("Error creating a Kafka adapter: %v\n", err)
	}

	log.Println("Starting service... version: ", cfg.Version)
	go consumerProcessor(ctx)

	ordModule := orderModule.New(ctx, cacheInstance, storage)
	apiServer := server.New(ctx, ordModule, &cfg.HTTPServer)

	log.Println("Starting server...")
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("Error starting server: %v", err)
		}
	}()

	// Завершение работы
	if err = gracefulShutdown(ctxCancel); err != nil {
		log.Fatalf("Fatal ERROR: %v", err)
	}
}

func gracefulShutdown(ctxCancel context.CancelFunc) error {
	const fn = "gracefulShutdown"

	// Канал для обработки сигнала завершения
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	log.Println("Shutting down gracefully...")
	ctxCancel()

	if kfkAdapter != nil {
		if err := kfkAdapter.Close(); err != nil {
			return fmt.Errorf("(%s) | Error closing Kafka adapter: %w", fn, err)
		}
	}

	if storage != nil {
		storage.Close()
	}

	time.Sleep(2 * time.Second)
	log.Println("Service stopped.")
	return nil
}

func cacheFill(ctx context.Context, limit int) error {
	const fn = "cacheFill"

	orders, err := storage.GetLastLimitOrders(ctx, limit)
	if err != nil {
		return fmt.Errorf("(%s) | Error getting last orders: %w", fn, err)
	} else if len(orders) == 0 {
		log.Printf("(%s) DB is empty!\n", fn)
		return nil
	}

	log.Printf("(%s) | The cache filling has started...\n", fn)
	for _, order := range orders {
		if !cacheInstance.Set(order.OrderUID, order) {
			return fmt.Errorf("(%s) | Error filling order: %v", fn, order.OrderUID)
		}
	}
	log.Printf("(%s) | The cache has been filled!\n", fn)
	return nil
}

func consumerProcessor(ctx context.Context) {
	const fn = "consumerStart"

	kfkAdapter.Start(ctx, func(order models.Order) error {
		log.Printf("(%s) Message received: %s\n", fn, order.OrderUID)
		if err := storage.SaveOrder(ctx, order); err != nil {
			log.Printf("(%s) | Error saving order: %v\n", fn, err)
			return err
		}
		if !cacheInstance.Set(order.OrderUID, order) {
			log.Printf("(%s) | Error caching order!\n", fn)
			return fmt.Errorf("(%s) | error caching order", fn)
		}
		return nil
	})
}
