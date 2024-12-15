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

var Queries = map[string]string{
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
		INSERT INTO orders (order_uid, delivery_id, payment_id, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
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
			ORDER BY id DESC
			LIMIT $1
		) AS subquery
		ORDER BY id ASC;
	`,
	"getDelivery": `
		SELECT *
		FROM deliveries
		WHERE id = $1
	`,
	"getPayment": `
		SELECT *
		FROM payments
		WHERE id = $1
	`,
	"getOrderItems": `
		SELECT i.*
		FROM items i
		JOIN order_items oi ON i.id = oi.item_id
		JOIN orders o ON oi.order_id = o.id
		WHERE o.id = $1;
	`,
}

type Storage struct {
	pool *pgxpool.Pool
}

func getPsqlConStr(db config.DataBase) string {
	// postgresql://<username>:<password>@<hostname>:<port>/<dbname>
	str := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		db.Username, db.Password, db.Host, db.Port, db.DbName)
	return str
}

func NewPgsql(ctx context.Context, db_cfg config.DataBase) (*Storage, error) {
	const fn = "NewPgsql"
	dsn := getPsqlConStr(db_cfg)

	pool, err := pgxpool.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to create connection pool: %w", fn, err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("(%s) | failed to ping database: %w", fn, err)
	}

	log.Println("Connected to PostgreSQL (using pool) successfully!")
	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() {
	if s.pool != nil {
		s.pool.Close()
		log.Println("DataBase connection pool closed!")
	}
}

// extractStructValues извлекает значения всех экспортируемых полей структуры
func extractStructValues(entity interface{}) ([]interface{}, error) {
	const fn = "extractStructValues"

	val := reflect.ValueOf(entity)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("(%s) | entity is not a struct or pointer to struct", fn)
	}

	var values []interface{}
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.CanInterface() {
			values = append(values, field.Interface())
		}
	}

	return values, nil
}

func (s *Storage) saveEntityRow(ctx context.Context, query string, entity interface{}) (interface{}, error) {
	const fn = "saveEntity"

	values, err := extractStructValues(entity)
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

func (s *Storage) saveDeliveryAndPayment(ctx context.Context, delivery models.Delivery, payment models.Payment) ([]interface{}, error) {
	const fn = "saveDeliveryAndPayment"

	var deliveryID, paymentID interface{}
	g, gCtx := errgroup.WithContext(ctx)
	defer gCtx.Done()

	g.Go(func() error {
		var err error
		deliveryID, err = s.saveEntityRow(gCtx, Queries["insertDelivery"], delivery)
		if err != nil {
			return fmt.Errorf("(%s) | failed to save Delivery: %w", fn, err)
		}
		return nil
	})

	g.Go(func() error {
		var err error
		paymentID, err = s.saveEntityRow(gCtx, Queries["insertPayment"], payment)
		if err != nil {
			return fmt.Errorf("(%s) | failed to save Payment: %w", fn, err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("(%s) | failed to insert delivery and payment: %w", fn, err)
	}

	log.Printf("(%s) | Delivery and Payment saved successfully!\n", fn)
	return []interface{}{deliveryID, paymentID}, nil
}

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
			itemID, err := s.saveEntityRow(gCtx, Queries["insertItem"], item)
			if err != nil {
				return fmt.Errorf("(%s) | failed to save Item: %w", fn, err)
			}
			if _, err := s.pool.Exec(gCtx, Queries["insertOrderItems"], orderUID, itemID); err != nil {
				return fmt.Errorf("(%s) | failed to insert OrderUID_ItemsID: %w", fn, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("(%s) | failed to insert items: %w", fn, err)
	}

	log.Printf("(%s) | Items saved successfully!\n", fn)
	return nil
}

func (s *Storage) SaveOrder(ctx context.Context, order models.Order) error {
	const fn = "SaveOrder"

	if exist, err := s.checkOrderExists(ctx, order.OrderUID); err != nil {
		return fmt.Errorf("(%s) | failed to check order by UID: %w", fn, err)
	} else if exist {
		return nil
	}

	devAndPayIDs, err := s.saveDeliveryAndPayment(ctx, order.Delivery, order.Payment)
	if err != nil {
		return fmt.Errorf("(%s) | failed to call saveDeliveryAndPayment: %w", fn, err)
	}

	if _, err := s.pool.Exec(ctx, Queries["insertOrder"],
		order.OrderUID,
		devAndPayIDs[0],
		devAndPayIDs[1],
		order.TrackNumber,
		order.Entry,
		order.Locale,
		order.InternalSignature,
		order.CustomerID,
		order.DeliveryService,
		order.Shardkey,
		order.SmID,
		order.DateCreated,
		order.OofShard,
	); err != nil {
		return fmt.Errorf("(%s) | failed to insert order: %w", fn, err)
	}

	if err := s.saveItems(ctx, order.OrderUID, order.Items); err != nil {
		return fmt.Errorf("(%s) | failed to call saveItems: %w", fn, err)
	}

	log.Printf("(%s) | Order saved with ID: %s\n", fn, order.OrderUID)
	return nil
}

func (s *Storage) checkOrderExists(ctx context.Context, orderUID string) (bool, error) {
	var existing string
	err := s.pool.QueryRow(ctx, Queries["checkOrderByUID"], orderUID).Scan(&existing)
	if err != nil && err.Error() != "no rows in result set" {
		return true, err
	} else if existing != "" {
		log.Printf("Order with UID %s already exists!\n", orderUID)
		return true, nil
	}
	return false, nil
}

func (s *Storage) getDelivery(ctx context.Context, deliveryID int) (models.Delivery, error) {
	const fn = "getDelivery"

	var id int
	var delivery models.Delivery
	err := s.pool.QueryRow(ctx, "getDelivery", deliveryID).Scan(
		&id,
		&delivery.Name,
		&delivery.Phone,
		&delivery.Zip,
		&delivery.City,
		&delivery.Address,
		&delivery.Region,
		&delivery.Email,
	)
	if err != nil {
		return models.Delivery{}, fmt.Errorf("(%s) | failed to get delivery: %w", fn, err)
	}
	log.Printf("(%s) | Delivery found with ID: %d", fn, deliveryID)
	return delivery, nil
}

func (s *Storage) getPayment(ctx context.Context, paymentID int) (models.Payment, error) {
	const fn = "getPayment"

	var id int
	var payment models.Payment
	err := s.pool.QueryRow(ctx, "getPayment", paymentID).Scan(
		&id,
		&payment.Transaction,
		&payment.RequestID,
		&payment.Currency,
		&payment.Provider,
		&payment.Amount,
		&payment.PaymentDT,
		&payment.Bank,
		&payment.DeliveryCost,
		&payment.GoodsTotal,
		&payment.CustomFee,
	)

	if err != nil {
		return models.Payment{}, fmt.Errorf("(%s) | failed to get payment: %w", fn, err)
	}
	log.Printf("(%s) | Payment found with ID: %d", fn, paymentID)
	return payment, nil
}

func (s *Storage) getOrderItems(ctx context.Context, orderID int) ([]models.Item, error) {
	const fn = "getOrderItems"

	var items []models.Item

	rows, err := s.pool.Query(ctx, "getOrderItems", orderID)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to get order items: %w", fn, err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var item models.Item
		err = rows.Scan(
			&id,
			&item.ChrtID,
			&item.TrackNumber,
			&item.Price,
			&item.RID,
			&item.Name,
			&item.Sale,
			&item.Size,
			&item.TotalPrice,
			&item.NMID,
			&item.Brand,
			&item.Status,
		)
		if err != nil {
			return nil, fmt.Errorf("(%s) | failed to scan row: %w", fn, err)
		}
		items = append(items, item)
	}

	log.Printf("(%s) | All items found by orderID: %d", fn, orderID)
	return items, nil
}

func (s *Storage) GetLastLimitOrders(ctx context.Context, limit int) ([]models.Order, error) {
	const fn = "GetLastLimitOrders"

	rows, err := s.pool.Query(ctx, "getLastLimitOrders", limit)
	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to execute query getLastLimitOrders: %w", fn, err)
	}
	defer rows.Close()

	orders := make([]models.Order, 0, limit)
	var tempOrders []tempOrder

	for rows.Next() {
		var tempOrder tempOrder
		if err := rows.Scan(
			&tempOrder.OrderID,
			&tempOrder.DeliveyID,
			&tempOrder.PaymentID,
			&tempOrder.Order.OrderUID,
			&tempOrder.Order.TrackNumber,
			&tempOrder.Order.Entry,
			&tempOrder.Order.Locale,
			&tempOrder.Order.InternalSignature,
			&tempOrder.Order.CustomerID,
			&tempOrder.Order.DeliveryService,
			&tempOrder.Order.Shardkey,
			&tempOrder.Order.SmID,
			&tempOrder.Order.DateCreated,
			&tempOrder.Order.OofShard,
		); err != nil {
			return nil, fmt.Errorf("(%s) | failed to scan row: %w", fn, err)
		}
		tempOrders = append(tempOrders, tempOrder)
	}
	rows.Close()

	for _, tempOrder := range tempOrders {
		tempOrder.Order.Delivery, err = s.getDelivery(ctx, tempOrder.DeliveyID)
		if err != nil {
			return nil, fmt.Errorf("(%s) | failed to get delivery: %w", fn, err)
		}
		tempOrder.Order.Payment, err = s.getPayment(ctx, tempOrder.PaymentID)
		if err != nil {
			return nil, fmt.Errorf("(%s) | failed to get payment: %w", fn, err)
		}
		tempOrder.Order.Items, err = s.getOrderItems(ctx, tempOrder.OrderID)
		if err != nil {
			return nil, fmt.Errorf("(%s) | failed to get order items: %w", fn, err)
		}
		orders = append(orders, tempOrder.Order)
	}

	if err != nil {
		return nil, fmt.Errorf("(%s) | failed to scan rows: %w", fn, err)
	}

	log.Printf("(%s) | %d orders found!", fn, len(orders))
	return orders, nil
}

type tempOrder struct {
	OrderID   int
	DeliveyID int
	PaymentID int
	Order     models.Order
}
