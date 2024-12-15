-- Down migration script

-- Drop join table for order items
DROP TABLE IF EXISTS order_items;

-- Drop table for orders
DROP TABLE IF EXISTS orders;

-- Drop table for items
DROP TABLE IF EXISTS items;

-- Drop table for payments
DROP TABLE IF EXISTS payments;

-- Drop table for deliveries
DROP TABLE IF EXISTS deliveries;

-- Drop pgcrypto extension
DROP EXTENSION IF EXISTS pgcrypto;