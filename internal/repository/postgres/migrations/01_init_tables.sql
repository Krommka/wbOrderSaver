DO $$
BEGIN
CREATE TABLE IF NOT EXISTS payment_providers (
    provider_id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL CHECK (name ~ '^[A-Za-z0-9 ]+$')
    );

CREATE TABLE IF NOT EXISTS delivery_services (
    service_id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL CHECK (name ~ '^[A-Za-z0-9 ]+$')
    );

CREATE TABLE IF NOT EXISTS brands (
    brand_id SERIAL PRIMARY KEY,
    name VARCHAR(255) UNIQUE NOT NULL CHECK (name ~ '^[A-Za-z0-9 ]+$')
    );

CREATE TABLE IF NOT EXISTS item_statuses (
    status_id INTEGER PRIMARY KEY,
    description VARCHAR(255) NOT NULL
    );

CREATE TABLE IF NOT EXISTS currencies (
    currency_id CHAR(3) PRIMARY KEY CHECK (currency_id ~ '^[A-Z]{3}$'),
    name VARCHAR(50) NOT NULL,
    symbol VARCHAR(5)
    );

CREATE TABLE IF NOT EXISTS orders (
    order_uid VARCHAR(20) PRIMARY KEY,
    track_number VARCHAR(255) NOT NULL,
    entry VARCHAR(50) NOT NULL,
    locale VARCHAR(10) NOT NULL CHECK (locale ~ '^[a-z]{2}$'),
    internal_signature VARCHAR(255) DEFAULT '',
    customer_id VARCHAR(255) NOT NULL,
    delivery_service_id INTEGER REFERENCES delivery_services(service_id),
    shardkey VARCHAR(10) NOT NULL,
    sm_id INTEGER NOT NULL CHECK (sm_id >= 0),
    date_created TIMESTAMP WITH TIME ZONE NOT NULL,
                               oof_shard VARCHAR(10) NOT NULL,
    CONSTRAINT valid_order_uid CHECK (order_uid ~ '^[a-f0-9]{20}$')
    );

CREATE TABLE IF NOT EXISTS delivery (
    order_uid VARCHAR(20) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL CHECK (name ~ '^[A-Za-z ]+$'),
    phone VARCHAR(50) NOT NULL CHECK (phone ~ '^\+[0-9]{7,15}$'),
    zip VARCHAR(50) NOT NULL CHECK (zip ~ '^[0-9]+$'),
    city VARCHAR(100) NOT NULL CHECK (city ~ '^[A-Za-z ]+$'),
    address VARCHAR(255) NOT NULL,
    region VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL CHECK (email ~* '^[A-Za-z0-9._%-]+@[A-Za-z0-9.-]+[.][A-Za-z]+$')
    );

CREATE TABLE IF NOT EXISTS payment (
    transaction VARCHAR(255) PRIMARY KEY REFERENCES orders(order_uid) ON DELETE CASCADE,
    request_id VARCHAR(255) DEFAULT '',
    currency_id CHAR(3) REFERENCES currencies(currency_id),
    provider_id INTEGER REFERENCES payment_providers(provider_id),
    amount INTEGER NOT NULL CHECK (amount >= 0),
    payment_dt BIGINT NOT NULL CHECK (payment_dt > 0),
    bank VARCHAR(100) NOT NULL,
    delivery_cost INTEGER NOT NULL CHECK (delivery_cost >= 0),
    goods_total INTEGER NOT NULL CHECK (goods_total >= 0),
    custom_fee INTEGER NOT NULL CHECK (custom_fee >= 0)
    );

CREATE TABLE IF NOT EXISTS items (
    id SERIAL PRIMARY KEY,
    chrt_id BIGINT NOT NULL UNIQUE CHECK (chrt_id > 0),
    track_number VARCHAR(255) NOT NULL,
    price INTEGER NOT NULL CHECK (price > 0),
    rid VARCHAR(255) NOT NULL CHECK (rid ~ '^[a-f0-9]{20}$'),
    name VARCHAR(255) NOT NULL,
    sale INTEGER NOT NULL CHECK (sale BETWEEN 0 AND 100),
    size VARCHAR(50) NOT NULL,
    total_price INTEGER NOT NULL CHECK (total_price >= 0),
    nm_id BIGINT NOT NULL CHECK (nm_id > 0),
    brand_id INTEGER REFERENCES brands(brand_id),
    status_id INTEGER REFERENCES item_statuses(status_id)
    );

CREATE TABLE IF NOT EXISTS order_items (
    order_uid VARCHAR(255) REFERENCES orders(order_uid) ON DELETE CASCADE,
    item_id BIGINT REFERENCES items(id) ON DELETE CASCADE,
    quantity INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    PRIMARY KEY (order_uid, item_id)
);

EXCEPTION WHEN others THEN
  RAISE NOTICE 'Ошибка при создании таблиц: %', SQLERRM;
END $$;