package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"log"
	"time"
	"wb_l0/internal/domain"

	"wb_l0/configs"
)

type Store struct {
	db *sql.DB
}

func NewStore(ctx context.Context, cfg *configs.Config) (*Store, error) {
	store := &Store{}

	err := store.Connect(ctx, *cfg)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	return store, nil
}

func (s *Store) Connect(ctx context.Context, cfg configs.Config) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled before connection: %w", err)
	}

	connConfig, err := pgx.ParseConfig(fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&connect_timeout=%d",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Name,
		cfg.DB.ConnectTimeout.Seconds(),
	))
	if err != nil {
		return fmt.Errorf("failed to parse connection config: %w", err)
	}

	var db *sql.DB

	retries := cfg.DB.Retries
	retryDelay := 5 * time.Second

	for i := 0; i < retries; i++ {
		if err = ctx.Err(); err != nil {
			return fmt.Errorf("%w: context cancelled during retries", err)
		}

		db, err = openConnection(ctx, connConfig)
		if err == nil {
			break
		}

		log.Printf("Retry %d/%d: Failed to connect to database: %v", i+1, retries, err)

		select {
		case <-time.After(retryDelay):

		case <-ctx.Done():
			return fmt.Errorf("connection cancelled during retry delay: %w", ctx.Err())
		}
	}
	if err != nil {
		return fmt.Errorf("failed to connect to database after %d retries: %w", retries, err)
	}

	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxIdleConns(5)
	db.SetConnMaxIdleTime(5 * time.Minute)

	s.db = db

	return nil
}

func openConnection(ctx context.Context, config *pgx.ConnConfig) (*sql.DB, error) {
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	connStr := stdlib.RegisterConnConfig(config)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	return db, nil
}

func (s *Store) Disconnect(ctx context.Context) error {
	if s.db == nil {
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- s.db.Close()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
		return nil
	}
}

func (s *Store) CreateOrder(ctx context.Context, order domain.Order) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature,
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
        ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.ShardKey, order.SMID, order.DateCreated, order.OOFShard,
	)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
        INSERT INTO delivery (
            order_uid, name, phone, zip, city, address, region, email
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        ON CONFLICT (order_uid) DO NOTHING`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,
	)
	if err != nil {
		return fmt.Errorf("failed to insert delivery: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
        INSERT INTO payment (
            transaction, request_id, currency_id, provider_id, amount, 
            payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
        ON CONFLICT (transaction) DO NOTHING`,
		order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDT,
		order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal,
		order.Payment.CustomFee,
	)
	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	for _, item := range order.Items {
		var brandID int
		err = tx.QueryRowContext(ctx, `
            SELECT brand_id FROM brands WHERE name = $1
        `, item.Brand).Scan(&brandID)

		if err != nil {
			err = tx.QueryRowContext(ctx, `
                INSERT INTO brands (name) VALUES ($1)
                ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
                RETURNING brand_id
            `, item.Brand).Scan(&brandID)
			if err != nil {
				return fmt.Errorf("failed to get/create brand: %w", err)
			}
		}

		_, err = tx.ExecContext(ctx, `
            INSERT INTO items (
                order_uid, chrt_id, track_number, price, rid, 
                name, sale, size, total_price, nm_id, brand_id, status_id
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
            ON CONFLICT (order_uid, chrt_id) DO NOTHING`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.RID,
			item.Name, item.Sale, item.Size, item.TotalPrice, item.NMID, brandID, item.Status,
		)
		if err != nil {
			return fmt.Errorf("failed to insert item %d: %w", item.ChrtID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
