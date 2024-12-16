package order

import (
	"context"
	"demo_service/internal/models"
	"fmt"
)

type Cache interface {
	Set(key string, value models.Order) bool
	Get(key string) (models.Order, bool)
}

type DB interface {
	SaveOrder(ctx context.Context, order models.Order) error
	GetOrderByUID(ctx context.Context, orderUID string) (models.Order, error)
}

type Order struct {
	ctx   context.Context
	cache Cache
	db    DB
}

func New(ctx context.Context, cache Cache, db DB) *Order {
	return &Order{
		ctx:   ctx,
		cache: cache,
		db:    db,
	}
}

// GetOrder
func (o *Order) GetOrder(ctx context.Context, orderUID string) (*models.Order, error) {
	order, ok := o.cache.Get(orderUID)
	if ok {
		return &order, nil
	}

	return o.saveOrderInCacheAndGetIt(ctx, orderUID)
}

func (o *Order) saveOrderInCacheAndGetIt(ctx context.Context, orderUID string) (*models.Order, error) {
	const fn = "saveOrderInCacheAndGetIt"

	order, err := o.db.GetOrderByUID(ctx, orderUID)
	if err != nil {
		return nil, fmt.Errorf("(%s) | %w", fn, err)
	}

	o.cache.Set(orderUID, order)
	return &order, nil
}
