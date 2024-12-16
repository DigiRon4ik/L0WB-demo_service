// Package db provides functions and methods for interacting with the PostgreSQL database.
//
// It manages the connection pool, executes queries, and handles database transactions for the service.
//
// This includes transactions such as storing orders, payments, deliveries and items, and retrieving orders.
package db

import (
	"context"
	"demo_service/internal/config"
	"demo_service/internal/models"
	"fmt"
	"log"
	"reflect"

	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/sync/errgroup"
)

var queries = map[string]string{
	"checkOrderByUID": `
		SELECT order_uid
		FROM orders
		WHERE order_uid = $1;
	`,
	"insertDelivery": `
		INSERT INTO deliveries (name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT ON CONSTRAINT unique_deliveries
		DO UPDATE SET id = deliveries.id
		RETURNING id;
	`,
	"insertPayment": `
		INSERT INTO payments (transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT ON CONSTRAINT unique_payments
		DO UPDATE SET id = payments.id
		RETURNING id;
	`,
	"insertItem": `
		INSERT INTO items (chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT ON CONSTRAINT unique_items
		DO UPDATE SET id = items.id
		RETURNING id;
	`,
	"insertOrder": `
		INSERT INTO orders (order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard, delivery_id, payment_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (order_uid)
		DO NOTHING;
	`,
	"insertOrderItems": `
		INSERT INTO order_items (order_uid, item_id)
		VALUES ($1, $2)
		ON CONFLICT ON CONSTRAINT unique_order_items
		DO NOTHING;
	`,
	"getLastLimitOrders": `
		SELECT *
		FROM (
			SELECT *
			FROM orders
			ORDER BY date_created DESC
			LIMIT $1
		) AS subquery
		ORDER BY date_created ASC;
	`,
	"getDelivery": `
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries
		WHERE id = $1
	`,
	"getPayment": `
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		FROM payments
		WHERE id = $1
	`,
	"getOrderItems": `
		SELECT i.chrt_id, i.track_number, i.price, i.rid, i.name, i.sale, i.size, i.total_price, i.nm_id, i.brand, i.status
		FROM items i
		JOIN order_items oi ON i.id = oi.item_id
		JOIN orders o ON oi.order_uid = o.order_uid
		WHERE o.order_uid = $1;
	`,
	"getOrderByUID": `
		SELECT *
		FROM orders
		WHERE order_uid = $1
	`,
}

// Storage holds the database connection pool for interacting with the PostgreSQL database.
type Storage struct {
	pool *pgxpool.Pool
}

// getPsqlConStr generates a PostgreSQL connection string
// based on the provided database configuration.
func getPsqlConStr(db config.DataBase) string {
	// postgresql://<username>:<password>@<hostname>:<port>/<dbname>
	str := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		db.Username, db.Password, db.Host, db.Port, db.DbName)
	return str
}

// New creates a new Storage instance, establishing a connection pool to the database
// and checking the connection by pinging the database.
func New(ctx context.Context, dbCfg config.DataBase) (*Storage, error) {
	const fn = "New"
	dsn := getPsqlConStr(dbCfg)

	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to create connection pool: %w", fn, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("(%s) | failed to ping database: %w", fn, err)
	}

	log.Println("Connected to DB (using pool) successfully!")
	return &Storage{pool: pool}, nil
}

// Close closes the database connection pool if it's open, logging the closure.
func (s *Storage) Close() {
	if s.pool != nil {
		s.pool.Close()
		log.Println("DataBase connection pool closed!")
	}
}

// extractStructFields extracts the fields of a struct as interfaces,
// optionally returning their addresses,
// skipping fields without a "db" tag or with a "-" tag,
// and ensuring the struct is a valid pointer or value.
func extractStructFields(entity interface{}, returnAddresses bool) ([]interface{}, error) {
	const fn = "extractStructFields"

	val := reflect.ValueOf(entity)

	if returnAddresses {
		if val.Kind() != reflect.Ptr || val.IsNil() {
			return nil, fmt.Errorf("(%s) | entity must be a non-nil pointer to a struct", fn)
		}
		val = val.Elem()
	} else {
		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("(%s) | entity is not a struct or pointer to struct", fn)
	}

	var fields []interface{}
	typ := val.Type()

	for i := 0; i < typ.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		dbTag := fieldType.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		if returnAddresses {
			if field.CanSet() {
				fields = append(fields, field.Addr().Interface())
			}
		} else {
			if field.CanInterface() {
				fields = append(fields, field.Interface())
			}
		}
	}

	return fields, nil
}

/*
	DB Saving Functions
*/

// saveEntityRow inserts a new row into the database based on the provided entity,
// extracting the struct fields, and returns the value of the generated field (e.g., ID).
func (s *Storage) saveEntityRow(ctx context.Context, query string, entity interface{}) (interface{}, error) {
	const fn = "saveEntityRow"

	values, err := extractStructFields(entity, false)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to extract struct values: %w", fn, err)
	}

	entityType := reflect.TypeOf(entity).Name()
	var returnField interface{}

	err = s.pool.QueryRow(ctx, query, values...).Scan(&returnField)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to insert %s: %w", fn, entityType, err)
	}

	log.Printf("(%s) | %s saved with ID: %d\n", fn, entityType, returnField)
	return returnField, nil
}

// saveDeliveryAndPayment concurrently saves the delivery and payment entities to the database,
// returning the generated IDs for both entities after successful insertion.
func (s *Storage) saveDeliveryAndPayment(ctx context.Context, delivery models.Delivery, payment models.Payment) ([]interface{}, error) {
	const fn = "saveDeliveryAndPayment"

	var deliveryID, paymentID interface{}
	g, gCtx := errgroup.WithContext(ctx)
	defer gCtx.Done()

	g.Go(func() error {
		var err error
		deliveryID, err = s.saveEntityRow(gCtx, queries["insertDelivery"], delivery)
		if err != nil {
			return fmt.Errorf("(%s) | failed to save Delivery: %w", fn, err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		paymentID, err = s.saveEntityRow(gCtx, queries["insertPayment"], payment)
		if err != nil {
			return fmt.Errorf("(%s) | failed to save Payment: %w", fn, err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("(%s) | failed to insert delivery and payment: %w", fn, err)
	}
	gCtx.Done()

	log.Printf("(%s) | Delivery and Payment saved successfully!\n", fn)
	return []interface{}{deliveryID, paymentID}, nil
}

// saveItems concurrently saves the items for an order and associates them with the given orderUID,
// ensuring that each item is inserted into the database and linked to the order.
func (s *Storage) saveItems(ctx context.Context, orderUID string, items []models.Item) error {
	const fn = "saveItems"

	if len(items) == 0 {
		log.Printf("(%s) | No items to save!\n", fn)
		return nil
	}

	g, gCtx := errgroup.WithContext(ctx)
	defer gCtx.Done()

	for _, item := range items {
		g.Go(func() error {
			itemID, err := s.saveEntityRow(gCtx, queries["insertItem"], item)
			if err != nil {
				return fmt.Errorf("(%s) | failed to save Item: %w", fn, err)
			}
			if _, err := s.pool.Exec(gCtx, queries["insertOrderItems"], orderUID, itemID); err != nil {
				return fmt.Errorf("(%s) | failed to insert OrderUID_ItemsID: %w", fn, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("(%s) | failed to insert items: %w", fn, err)
	}
	gCtx.Done()

	log.Printf("(%s) | Items saved successfully!\n", fn)
	return nil
}

// SaveOrder checks if an order with the given UID already exists, and if not,
// saves the order along with its associated delivery, payment, and items to the database.
func (s *Storage) SaveOrder(ctx context.Context, order models.Order) error {
	const fn = "SaveOrder"

	if exist, err := s.checkOrderExists(ctx, order.OrderUID); err != nil {
		return fmt.Errorf("(%s) | failed to check order by UID: %w", fn, err)
	} else if exist {
		return nil
	}

	delAndPayIDs, err := s.saveDeliveryAndPayment(ctx, order.Delivery, order.Payment)
	if err != nil {
		return fmt.Errorf("(%s) | failed to call saveDeliveryAndPayment: %w", fn, err)
	}

	values, err := extractStructFields(order, false)
	if err != nil {
		return fmt.Errorf("(%s) | failed to extract: %w", fn, err)
	}
	values = append(values, delAndPayIDs...)
	if _, err := s.pool.Exec(ctx, queries["insertOrder"], values...); err != nil {
		return fmt.Errorf("(%s) | failed to insert order: %w", fn, err)
	}

	if err := s.saveItems(ctx, order.OrderUID, order.Items); err != nil {
		return fmt.Errorf("(%s) | failed to call saveItems: %w", fn, err)
	}

	log.Printf("(%s) | Order saved with ID: %s\n", fn, order.OrderUID)
	return nil
}

// checkOrderExists checks if an order with the given orderUID already exists in the database
// and returns true if it does, or false if it does not, along with any potential errors.
func (s *Storage) checkOrderExists(ctx context.Context, orderUID string) (bool, error) {
	var existing string
	err := s.pool.QueryRow(ctx, queries["checkOrderByUID"], orderUID).Scan(&existing)
	if err != nil && err.Error() != "no rows in result set" {
		return true, err
	} else if existing != "" {
		log.Printf("Order with UID %s already exists!\n", orderUID)
		return true, nil
	}
	return false, nil
}

/*
	DB Getting Functions
*/

// getEntityRow executes a query to fetch a single row from the database,
// scans the result into the provided entity, and logs the success or failure of the operation.
func (s *Storage) getEntityRow(ctx context.Context, query string, queryValues []interface{}, entity interface{}) error {
	const fn = "getEntityRow"

	scanArgs, err := extractStructFields(entity, true)
	if err != nil {
		return fmt.Errorf("(%s) | failed to extract scan args: %w", fn, err)
	}

	entityType := reflect.TypeOf(entity)

	err = s.pool.QueryRow(ctx, query, queryValues...).Scan(scanArgs...)
	if err != nil {
		return fmt.Errorf("(%s) | failed to scan %s: %w", fn, entityType, err)
	}

	log.Printf("(%s) | %s found!\n", fn, entityType)
	return nil
}

// getOrderItems retrieves all items for a given orderUID from the database,
// scans the results concurrently, and returns the list of items associated with the order.
func (s *Storage) getOrderItems(ctx context.Context, orderUID string) ([]models.Item, error) {
	const fn = "getOrderItems"

	var items []models.Item

	rows, err := s.pool.Query(ctx, queries["getOrderItems"], orderUID)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to get order items: %w", fn, err)
	}
	defer rows.Close()

	g, gCtx := errgroup.WithContext(ctx)
	defer gCtx.Done()

	for rows.Next() {
		g.Go(func() error {
			var item models.Item
			scanArgs, err := extractStructFields(&item, true)
			if err != nil {
				return fmt.Errorf("(%s) | failed to extract: %w", fn, err)
			}
			if err := rows.Scan(scanArgs...); err != nil {
				return fmt.Errorf("(%s) | failed to scan row: %w", fn, err)
			}
			items = append(items, item)
			return nil
		})
	}
	rows.Close()

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("(%s) | failed to get items: %w", fn, err)
	}
	gCtx.Done()

	log.Printf("(%s) | All items found by orderUID: %v", fn, orderUID)
	return items, nil
}

// GetLastLimitOrders retrieves the last 'limit' orders from the database,
// concurrently finalizes each order by fetching related delivery, payment, and items,
// and returns the populated (done) orders.
func (s *Storage) GetLastLimitOrders(ctx context.Context, limit int) ([]models.Order, error) {
	const fn = "GetLastLimitOrders"

	rows, err := s.pool.Query(ctx, queries["getLastLimitOrders"], limit)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to execute query getLastLimitOrders: %w", fn, err)
	}
	defer rows.Close()

	orders := make([]models.Order, 0, limit)
	g, gCtx := errgroup.WithContext(ctx)
	defer gCtx.Done()
	for rows.Next() {
		g.Go(func() error {
			var order models.Order
			var deliveryID, paymentID int

			scanArgs, err := extractStructFields(&order, true)
			if err != nil {
				return fmt.Errorf("(%s) | failed to extract: %w", fn, err)
			}
			scanArgs = append(scanArgs, &deliveryID, &paymentID)
			if err := rows.Scan(scanArgs...); err != nil {
				return fmt.Errorf("(%s) | failed to scan row: %w", fn, err)
			}

			if err := s.finalizeOrder(gCtx, &order, deliveryID, paymentID); err != nil {
				return fmt.Errorf("(%s) | failed to call finalizeOrder: %w", fn, err)
			}

			orders = append(orders, order)
			return nil
		})
	}
	rows.Close()

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("(%s) | Failed read order rows: %w", fn, err)
	}
	gCtx.Done()

	log.Printf("(%s) | %d orders found!", fn, len(orders))
	return orders, nil
}

// GetOrderByUID retrieves the order by its UID from the database, scans the relevant data,
// and finalizes the order by adding shipping, payment, and item information, then returns it.
func (s *Storage) GetOrderByUID(ctx context.Context, orderUID string) (models.Order, error) {
	const fn = "GetOrder"

	var order models.Order
	var deliveryID, paymentID int

	scanArgs, err := extractStructFields(&order, true)
	if err != nil {
		return models.Order{}, fmt.Errorf("(%s) | failed to extract: %w", fn, err)
	}
	scanArgs = append(scanArgs, &deliveryID, &paymentID)
	if err := s.pool.QueryRow(ctx, queries["getOrderByUID"], orderUID).Scan(scanArgs...); err != nil {
		return models.Order{}, fmt.Errorf("(%s) | failed to scan row: %w", fn, err)
	}
	if err := s.finalizeOrder(ctx, &order, deliveryID, paymentID); err != nil {
		return models.Order{}, fmt.Errorf("(%s) | failed to call finalizeOrder: %w", fn, err)
	}
	return order, nil
}

// finalizeOrder concurrently fetches the delivery, payment, and items for the order
// and populates the respective fields in the order object, handling errors during the process.
func (s *Storage) finalizeOrder(ctx context.Context, order *models.Order, dID, pID int) error {
	const fn = "finalizeOrder"

	g, gCtx := errgroup.WithContext(ctx)
	defer gCtx.Done()

	g.Go(func() error {
		err := s.getEntityRow(gCtx, queries["getDelivery"], []interface{}{dID}, &order.Delivery)
		if err != nil {
			return fmt.Errorf("(%s) | failed to get Delivery: %w", fn, err)
		}
		return nil
	})

	g.Go(func() error {
		err := s.getEntityRow(gCtx, queries["getPayment"], []interface{}{pID}, &order.Payment)
		if err != nil {
			return fmt.Errorf("(%s) | failed to get Payment: %w", fn, err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		order.Items, err = s.getOrderItems(gCtx, order.OrderUID)
		if err != nil {
			return fmt.Errorf("(%s) | failed to get order items: %w", fn, err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("(%s) | Order finalization error: %w", fn, err)
	}
	gCtx.Done()
	return nil
}
