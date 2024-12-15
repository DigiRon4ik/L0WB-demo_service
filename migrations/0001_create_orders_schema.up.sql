-- Create pgcrypto extension
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- Create table for deliveries with unique constraint
CREATE TABLE IF NOT EXISTS deliveries (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    zip VARCHAR(20) NOT NULL,
    city VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    region VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    CONSTRAINT unique_deliveries UNIQUE (
        name,
        phone,
        zip,
        city,
        address,
        region,
        email
    )
);

-- Create table for payments with unique constraint
CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    transaction VARCHAR(255) NOT NULL,
    request_id VARCHAR(255),
    currency VARCHAR(10) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    amount INTEGER NOT NULL,
    payment_dt BIGINT NOT NULL,
    bank VARCHAR(50) NOT NULL,
    delivery_cost INTEGER NOT NULL,
    goods_total INTEGER NOT NULL,
    custom_fee INTEGER NOT NULL,
    CONSTRAINT unique_payments UNIQUE (
        transaction,
        request_id,
        currency,
        provider,
        amount,
        payment_dt,
        bank,
        delivery_cost,
        goods_total,
        custom_fee
    )
);

-- Create table for items with unique constraint
CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    chrt_id INTEGER NOT NULL,
    track_number VARCHAR(255) NOT NULL,
    price INTEGER NOT NULL,
    rid VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    sale INTEGER NOT NULL,
    size VARCHAR(50) NOT NULL,
    total_price INTEGER NOT NULL,
    nm_id INTEGER NOT NULL,
    brand VARCHAR(255) NOT NULL,
    status INTEGER NOT NULL,
    CONSTRAINT unique_items UNIQUE (
        chrt_id,
        track_number,
        price,
        rid,
        name,
        sale,
        size,
        total_price,
        nm_id,
        brand,
        status
    )
);

-- Create table for orders with unique constraint
CREATE TABLE IF NOT EXISTS orders (
    order_uid VARCHAR(20) PRIMARY KEY,
    delivery_id INTEGER NOT NULL REFERENCES deliveries (id) ON DELETE CASCADE,
    payment_id INTEGER NOT NULL REFERENCES payments (id) ON DELETE CASCADE,
    track_number VARCHAR(255) NOT NULL,
    entry VARCHAR(50) NOT NULL,
    locale VARCHAR(10) NOT NULL,
    internal_signature VARCHAR(255),
    customer_id VARCHAR(255) NOT NULL,
    delivery_service VARCHAR(255) NOT NULL,
    shardkey VARCHAR(50) NOT NULL,
    sm_id INTEGER NOT NULL,
    date_created TIMESTAMP NOT NULL,
    oof_shard VARCHAR(50) NOT NULL
);

-- Create join table for order items with unique constraint
CREATE TABLE IF NOT EXISTS order_items (
    id SERIAL PRIMARY KEY,
    order_uid VARCHAR(20) NOT NULL REFERENCES orders (order_uid) ON DELETE CASCADE,
    item_id INTEGER NOT NULL REFERENCES items (id) ON DELETE CASCADE,
    CONSTRAINT unique_order_items UNIQUE (order_uid, item_id)
);