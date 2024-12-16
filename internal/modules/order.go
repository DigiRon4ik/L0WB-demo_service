// Package order provides functionality for managing orders.
// It includes operations for retrieving orders from the cache or database,
// and for saving them in the cache. The package utilizes a caching layer
// to optimize performance by reducing redundant database queries.
package order

import (
	"context"
	"demo_service/internal/models"
	"fmt"
)

// Cache interface defines methods for working with the cache.
type Cache interface {
	Set(key string, value models.Order) bool
	Get(key string) (models.Order, bool)
}

// DB interface defines methods for interacting with the order database.
type DB interface {
	SaveOrder(ctx context.Context, order models.Order) error
	GetOrderByUID(ctx context.Context, orderUID string) (models.Order, error)
}

// Order struct represents the order of the module with context, cache and database.
type Order struct {
	ctx   context.Context
	cache Cache
	db    DB
}

// New creates a new module Order object with the specified context, cache and database.
func New(ctx context.Context, cache Cache, db DB) *Order {
	return &Order{
		ctx:   ctx,
		cache: cache,
		db:    db,
	}
}

// GetOrder retrieves the order by its unique ID (orderUID).
// It first checks if the order is in the cache. If found, it returns it.
// Otherwise, it fetches the order from the database and stores it in the cache.
func (o *Order) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	order, ok := o.cache.Get(orderUID)
	if ok {
		return &order, nil
	}

	return o.saveOrderInCacheAndGetIt(ctx, orderUID)
}

// saveOrderInCacheAndGetIt fetches the order from the database by its unique ID (orderUID),
// stores it in the cache, and then returns it.
func (o *Order) saveOrderInCacheAndGetIt(ctx context.Context, orderUID string) (*models.Order, error) {
	const fn = "saveOrderInCacheAndGetIt"

	order, err := o.db.GetOrderByUID(ctx, orderUID)
	if err != nil {
		return nil, fmt.Errorf("(%s) | %w", fn, err)
	}

	o.cache.Set(orderUID, order)
	return &order, nil
}
