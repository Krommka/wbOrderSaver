package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
	"wb_l0/configs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

type Store struct {
	db  *sql.DB
	log *slog.Logger
}

func NewStore(ctx context.Context, cfg *configs.Config, log *slog.Logger) (*Store, error) {
	store := &Store{log: log}

	err := store.Connect(ctx, *cfg)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	return store, nil
}

func (s *Store) Connect(ctx context.Context, cfg configs.Config) error {
	s.log.Info("Connecting to database",
		"host", cfg.DB.Host,
		"port", cfg.DB.Port,
		"database", cfg.DB.Name,
		"user", cfg.DB.User,
		"timeout", cfg.DB.ConnectTimeout,
		"retries", cfg.DB.Retries)
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
		int(cfg.DB.ConnectTimeout.Seconds()),
	))
	if err != nil {
		return fmt.Errorf("failed to parse connection config: %w", err)
	}

	var db *sql.DB

	retries := cfg.DB.Retries
	retryDelay := cfg.DB.ConnectTimeout

	for i := 0; i < retries; i++ {
		if err = ctx.Err(); err != nil {
			return fmt.Errorf("%w: context cancelled during retries", err)
		}

		db, err = openConnection(ctx, cfg.DB.ConnectTimeout, connConfig)
		if err == nil {
			break
		}
		s.log.Error("failed to connect to database", "retry", i+1, "retries", retries, "error", err)

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

func openConnection(ctx context.Context, timeout time.Duration, config *pgx.ConnConfig) (*sql.DB, error) {
	pingCtx, cancel := context.WithTimeout(ctx, timeout)
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
