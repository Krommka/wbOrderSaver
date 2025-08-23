package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"wb_l0/internal/domain"
)

func (s *Store) SaveOrder(ctx context.Context, order *domain.Order) error {
	startTime := time.Now()
	s.log.Info("Database operation started",
		"operation", "SaveOrder",
		"order_uid", order.OrderUID,
		"items_count", len(order.Items),
	)

	exists, err := s.checkOrderExists(ctx, order.OrderUID)
	if err != nil {
		s.log.Error("Failed to check order existence",
			"order_uid", order.OrderUID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to check order existence: %w", err)
	}
	if exists {
		s.log.Warn("Order already exists - skipping processing",
			"order_uid", order.OrderUID,
			"action", "skip_duplicate",
		)
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		s.log.Error("Failed to begin transaction",
			"order_uid", order.OrderUID,
			"error", err.Error(),
			"operation", "begin_transaction",
		)
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	deliveryServiceID, err := s.getOrCreateDeliveryServiceID(ctx, tx, order)
	if err != nil {
		s.log.Error("Failed to create delivery service",
			"delivery_service", order.DeliveryService,
			"order_uid", order.OrderUID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to create delivery service: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature,
            customer_id, delivery_service_id, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, deliveryServiceID, order.ShardKey, order.SMID, order.DateCreated, order.OOFShard,
	)
	if err != nil {
		s.log.Error("Failed to insert order",
			"order_uid", order.OrderUID,
			"error", err.Error(),
			"table", "orders",
		)
		return fmt.Errorf("failed to insert order: %w", err)
	}
	s.log.Debug("Order inserted successfully",
		"order_uid", order.OrderUID,
		"table", "orders",
	)

	_, err = tx.ExecContext(ctx, `
        INSERT INTO delivery (
            order_uid, name, phone, zip, city, address, region, email
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
	)
	if err != nil {
		s.log.Error("Failed to insert delivery",
			"order_uid", order.OrderUID,
			"error", err.Error(),
			"table", "delivery",
		)
		return fmt.Errorf("failed to insert delivery: %w", err)
	}

	paymentProviderID, err := s.getOrCreatePaymentProviderID(ctx, tx, order)
	if err != nil {
		s.log.Error("Failed to create payment provider",
			"payment_provider", order.Payment.Provider,
			"order_uid", order.OrderUID,
			"error", err.Error(),
		)
		return fmt.Errorf("failed to create delivery service: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
        INSERT INTO payment (
            transaction, request_id, currency_id, provider_id, amount, 
            payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (transaction) DO NOTHING`,
		order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		paymentProviderID, order.Payment.Amount, order.Payment.PaymentDT,
		order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal,
		order.Payment.CustomFee,
	)
	if err != nil {
		s.log.Error("Failed to insert payment",
			"order_uid", order.OrderUID,
			"error", err.Error(),
			"table", "payment",
		)
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	for i, item := range order.Items {
		brandID, err := s.getOrCreateBrandID(ctx, tx, item)
		if err != nil {
			s.log.Error("Failed to create brand",
				"brand", order.Items[i].Brand,
				"order_uid", order.OrderUID,
				"error", err.Error(),
			)
			return fmt.Errorf("failed to create brand: %w", err)
		}

		var itemID int
		err = tx.QueryRowContext(ctx, `
            INSERT INTO items (
                chrt_id, track_number, price, rid, 
                name, sale, size, total_price, nm_id, brand_id, status_id
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
            ON CONFLICT (chrt_id) DO UPDATE SET
                track_number = EXCLUDED.track_number,
                price = EXCLUDED.price,
                rid = EXCLUDED.rid,
                name = EXCLUDED.name,
                sale = EXCLUDED.sale,
                size = EXCLUDED.size,
                total_price = EXCLUDED.total_price,
                nm_id = EXCLUDED.nm_id,
                brand_id = EXCLUDED.brand_id,
                status_id = EXCLUDED.status_id
                RETURNING id`,
			item.ChrtID, item.TrackNumber, item.Price, item.RID,
			item.Name, item.Sale, item.Size, item.TotalPrice, item.NMID, brandID, item.Status,
		).Scan(&itemID)
		if err != nil {
			s.log.Error("Failed to insert item",
				"order_uid", order.OrderUID,
				"chrt_id", item.ChrtID,
				"error", err.Error(),
				"table", "items",
			)
			return fmt.Errorf("failed to insert/update item %d: %w", item.ChrtID, err)
		}

		_, err = tx.ExecContext(ctx, `
            INSERT INTO order_items (order_uid, item_id, quantity)
            VALUES ($1, $2, 1)
            ON CONFLICT (order_uid, item_id) DO UPDATE SET
                quantity = order_items.quantity + 1`,
			order.OrderUID, itemID,
		)
		if err != nil {
			s.log.Error("Failed to create order-item link",
				"order_uid", order.OrderUID,
				"item_id", itemID,
				"chrt_id", item.ChrtID,
				"error", err.Error(),
				"table", "order_items",
			)
			return fmt.Errorf("failed to create order-item link %d: %w", item.ChrtID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		s.log.Error("Failed to commit transaction",
			"order_uid", order.OrderUID,
			"error", err.Error(),
			"operation", "commit",
		)
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("Order saved successfully",
		"order_uid", order.OrderUID,
		"total_processing_time_ms", time.Since(startTime).Milliseconds(),
		"status", "completed",
	)
	return nil
}

func (s *Store) GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error) {
	startTime := time.Now()

	s.log.Info("Database query started",
		"operation", "GetOrderByUID",
		"order_uid", orderUID,
		"query_type", "read",
	)
	query := `
        SELECT 
            o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature,
            o.customer_id, ds.name as delivery_service, o.shardkey, o.sm_id, 
            o.date_created, o.oof_shard,
            d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
            p.transaction, p.request_id, c.currency_id, pp.name as provider_name,
            p.amount, p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
        FROM orders o
        JOIN delivery d ON o.order_uid = d.order_uid
        JOIN payment p ON o.order_uid = p.transaction
        JOIN delivery_services ds ON o.delivery_service_id = ds.service_id
        JOIN payment_providers pp ON p.provider_id = pp.provider_id
        JOIN currencies c ON p.currency_id = c.currency_id
        WHERE o.order_uid = $1
    `

	var order domain.Order
	var paymentProvider, currency string

	s.log.Debug("Executing SQL query",
		"order_uid", orderUID,
		"query", "GetOrder_main",
		"tables", []string{"orders", "delivery", "payment", "delivery_services", "payment_providers", "currencies"},
	)

	err := s.db.QueryRowContext(ctx, query, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerID, &order.DeliveryService, &order.ShardKey, &order.SMID,
		&order.DateCreated, &order.OOFShard,
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City,
		&order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,
		&order.Payment.Transaction, &order.Payment.RequestID, &currency, &paymentProvider,
		&order.Payment.Amount, &order.Payment.PaymentDT, &order.Payment.Bank,
		&order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			s.log.Warn("Order not found",
				"order_uid", orderUID,
				"error", "not_found",
				"query_time_ms", time.Since(startTime).Milliseconds(),
			)
			return nil, domain.ErrRecordNotFound
		}
		s.log.Error("Failed to execute query",
			"order_uid", orderUID,
			"error", err.Error(),
			"error_type", "database_query",
			"query_time_ms", time.Since(startTime).Milliseconds(),
		)
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	order.Payment.Provider = paymentProvider
	order.Payment.Currency = currency

	s.log.Debug("Main order data retrieved",
		"order_uid", orderUID,
		"customer_id", order.CustomerID,
		"delivery_service", order.DeliveryService,
		"amount", order.Payment.Amount,
		"query_time_ms", time.Since(startTime).Milliseconds(),
	)
	itemsStartTime := time.Now()
	items, err := s.getOrderItems(ctx, orderUID)
	if err != nil {
		s.log.Error("Failed to get order items",
			"order_uid", orderUID,
			"error", err.Error(),
			"error_type", "items_query",
			"items_query_time_ms", time.Since(itemsStartTime).Milliseconds(),
		)
		return nil, fmt.Errorf("failed to get order items: %w", err)
	}
	order.Items = items
	s.log.Info("Order retrieved successfully",
		"order_uid", orderUID,
		"items_count", len(items),
		"total_query_time_ms", time.Since(startTime).Milliseconds(),
		"main_query_time_ms", time.Since(startTime).Milliseconds()-time.Since(itemsStartTime).Milliseconds(),
		"items_query_time_ms", time.Since(itemsStartTime).Milliseconds(),
		"status", "success",
	)
	return &order, nil
}

func (s *Store) getOrderItems(ctx context.Context, orderUID string) ([]domain.Item, error) {
	itemsStartTime := time.Now()

	s.log.Debug("Fetching order items",
		"order_uid", orderUID,
		"operation", "getOrderItems",
	)
	query := `
        SELECT 
            i.chrt_id, i.track_number, i.price, i.rid, i.name, i.sale, i.size,
            i.total_price, i.nm_id, b.name as brand_name, i.status_id,
            oi.quantity
        FROM order_items oi
        JOIN items i ON oi.item_id = i.id
        JOIN brands b ON i.brand_id = b.brand_id
        WHERE oi.order_uid = $1
    `

	rows, err := s.db.QueryContext(ctx, query, orderUID)
	if err != nil {
		s.log.Error("Failed to query order items",
			"order_uid", orderUID,
			"error", err.Error(),
			"query", "getOrderItems",
		)
		return nil, fmt.Errorf("failed to query order items: %w", err)
	}
	defer rows.Close()

	var items []domain.Item
	var itemsProcessed int

	for rows.Next() {
		var item domain.Item
		var brand string
		var quantity int

		err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.RID, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NMID, &brand, &item.Status,
			&quantity,
		)
		if err != nil {
			s.log.Error("Failed to scan item row",
				"order_uid", orderUID,
				"error", err.Error(),
				"operation", "scan_row",
			)
			continue // Пропускаем проблемную строку, но продолжаем обработку
		}

		item.Brand = brand
		items = append(items, item)
		itemsProcessed++
	}

	if err := rows.Err(); err != nil {
		s.log.Error("Error iterating items",
			"order_uid", orderUID,
			"error", err.Error(),
			"operation", "rows_iteration",
		)
		return nil, fmt.Errorf("error iterating items: %w", err)
	}

	s.log.Debug("Order items retrieved",
		"order_uid", orderUID,
		"items_count", itemsProcessed,
		"query_time_ms", time.Since(itemsStartTime).Milliseconds(),
	)

	return items, nil
}

func (s *Store) DeleteOrder(ctx context.Context, orderUID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx, `
        SELECT EXISTS(SELECT 1 FROM orders WHERE order_uid = $1)
    `, orderUID).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check order existence: %w", err)
	}

	if !exists {
		return fmt.Errorf("order with UID %s not found", orderUID)
	}

	_, err = tx.ExecContext(ctx, `
        DELETE FROM orders WHERE order_uid = $1
    `, orderUID)

	if err != nil {
		return fmt.Errorf("failed to delete order: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (s *Store) getOrCreateDeliveryServiceID(ctx context.Context, tx *sql.Tx, order *domain.Order) (int, error) {
	var deliveryServiceID int
	err := tx.QueryRowContext(ctx, `
            SELECT service_id FROM delivery_services WHERE name = $1
        `, order.DeliveryService).Scan(&deliveryServiceID)

	if err != nil {
		s.log.Debug("Delivery service not found, creating new",
			"delivery_service", order.DeliveryService,
			"order_uid", order.OrderUID,
		)
		err = tx.QueryRowContext(ctx, `
                INSERT INTO delivery_services (name) VALUES ($1)
                ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
                RETURNING service_id
            `, order.DeliveryService).Scan(&deliveryServiceID)
		if err != nil {
			return 0, fmt.Errorf("failed to get/create DeliveryService: %w", err)
		}
	}
	return deliveryServiceID, nil
}

func (s *Store) getOrCreatePaymentProviderID(ctx context.Context, tx *sql.Tx, order *domain.Order) (int, error) {
	var paymentProviderID int
	err := tx.QueryRowContext(ctx, `
            SELECT provider_id FROM payment_providers WHERE name = $1
        `, order.Payment.Provider).Scan(&paymentProviderID)

	if err != nil {
		err = tx.QueryRowContext(ctx, `
                INSERT INTO payment_providers (name) VALUES ($1)
                ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
                RETURNING provider_id
            `, order.Payment.Provider).Scan(&paymentProviderID)
		if err != nil {
			return 0, fmt.Errorf("failed to get/create PaymentProvider: %w", err)
		}
	}
	return paymentProviderID, nil
}

func (s *Store) getOrCreateBrandID(ctx context.Context, tx *sql.Tx, item domain.Item) (int, error) {
	var brandID int

	err := tx.QueryRowContext(ctx, `
            SELECT brand_id FROM brands WHERE name = $1
        `, item.Brand).Scan(&brandID)

	if err != nil {
		err = tx.QueryRowContext(ctx, `
                INSERT INTO brands (name) VALUES ($1)
                ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
                RETURNING brand_id
            `, item.Brand).Scan(&brandID)
		if err != nil {
			return 0, fmt.Errorf("failed to get/create brand: %w", err)
		}
	}
	return brandID, nil
}

func (s *Store) checkOrderExists(ctx context.Context, orderUID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
        SELECT EXISTS(SELECT 1 FROM orders WHERE order_uid = $1)
    `, orderUID).Scan(&exists)

	if err != nil {
		return false, err
	}

	return exists, nil
}
