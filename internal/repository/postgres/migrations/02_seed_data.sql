DO $$
BEGIN
INSERT INTO item_statuses (status_id, description) VALUES
    (200, 'Created'),
    (202, 'Approved'),
    (300, 'Sale'),
    (400, 'Not available')
    ON CONFLICT (status_id) DO UPDATE SET description = EXCLUDED.description;

INSERT INTO brands (name) VALUES
    ('Vivienne Sabo'),
    ('Nike'),
    ('Guess'),
    ('Uniqlo')
    ON CONFLICT (name) DO NOTHING;

INSERT INTO payment_providers (name) VALUES
    ('wbpay'),
    ('mir'),
    ('ukassa')
    ON CONFLICT (name) DO NOTHING;

INSERT INTO delivery_services (name) VALUES
    ('meest'),
    ('pochta'),
    ('sdek')
    ON CONFLICT (name) DO NOTHING;

INSERT INTO currencies (currency_id, name, symbol) VALUES
    ('USD', 'US Dollar', '$'),
    ('EUR', 'Euro', '€'),
    ('RUB', 'Russian Ruble', '₽'),
    ('CNY', 'Chinese Yuan', '¥')
    ON CONFLICT (currency_id) DO UPDATE
        SET name = EXCLUDED.name, symbol = EXCLUDED.symbol;

EXCEPTION WHEN others THEN
  RAISE NOTICE 'Ошибка при заполнении справочников: %', SQLERRM;
END $$;