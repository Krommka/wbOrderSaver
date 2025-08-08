package postgres

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"Go_Team00.ID_376234-Team_TL_barievel/configs"
	"Go_Team00.ID_376234-Team_TL_barievel/db"
	"Go_Team00.ID_376234-Team_TL_barievel/internal/entities"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Store struct {
	db *gorm.DB
}

func NewStore(ctx context.Context, cfg configs.Config) (*Store, error) {
	store := &Store{}

	err := store.Connect(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect failed: %w", err)
	}
	return store, nil
}

func (store *Store) Connect(ctx context.Context, cfg configs.Config) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: context cancelled before connection", db.ErrDBConnection)
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=%s",
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Name,
		cfg.DB.ConnectTimeout,
	)

	var dataBase *gorm.DB
	var err error

	retries := cfg.DB.Retries
	retryDelay := 5 * time.Second

	for i := 0; i < retries; i++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("%w: context cancelled during retries", db.ErrDBConnection)
		}

		dataBase, err = openConnection(ctx, dsn)
		if err == nil {
			break
		}

		log.Printf("Retry %d/%d: Failed to connect to database: %v", i+1, retries, err)

		select {
		case <-time.After(retryDelay):

		case <-ctx.Done():
			return fmt.Errorf("%w: connection cancelled during retry delay", db.ErrTimeout)
		}
	}
	if err != nil {
		return fmt.Errorf("%w: failed to connect to database after %d retries: %v", db.ErrDBConnection, retries, err)
	}

	sqlDB, err := dataBase.DB()
	if err != nil {
		return fmt.Errorf("%w: failed to retrieve sql.DB object: %v", db.ErrDBConnection, err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	store.db = dataBase

	if err := store.Migrate(); err != nil {
		return err
	}

	return nil
}

func openConnection(ctx context.Context, dsn string) (*gorm.DB, error) {
	if ctx.Err() != nil {
		return nil, fmt.Errorf("%w: context cancelled before connection", db.ErrTimeout)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resultChan := make(chan struct {
		db  *gorm.DB
		err error
	})

	go func() {

		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		resultChan <- struct {
			db  *gorm.DB
			err error
		}{db, err}
	}()

	select {
	case <-pingCtx.Done():
		return nil, fmt.Errorf("%w: context cancelled during connection", db.ErrTimeout)
	case result := <-resultChan:
		if result.err != nil {
			return nil, fmt.Errorf("%w: failed to initialize database object: %v", db.ErrDBConnection, result.err)
		}
		sqlDB, err := result.db.DB()
		if err != nil {
			return nil, fmt.Errorf("%w: failed to check database connection: %v", db.ErrDBConnection, err)
		}
		if err := sqlDB.PingContext(pingCtx); err != nil {
			return nil, fmt.Errorf("%w: database ping failed: %v", db.ErrDBConnection, err)
		}
		return result.db, nil
	}
}

func (store *Store) Migrate() error {
	if err := store.db.AutoMigrate(&db.EntryDB{}); err != nil {
		return fmt.Errorf("error migrating database: %v", err)
	}
	return nil
}

// Disconnect Close connection
func (store *Store) Disconnect(ctx context.Context) error {
	if store.db == nil {
		return fmt.Errorf("store is nil")
	}

	sqlDB, err := store.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	done := make(chan error, 1)

	go func() {
		done <- sqlDB.Close()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("database shutdown timed out: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to shutdown database: %v", err)
		}
	}
	return nil
}

// GetAnomaly Send SELECT statement with session_id, return Entry
func (store *Store) GetAnomaly(ctx context.Context, sessionId string) ([]entities.Entry, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: context error before query", db.ErrDBQuery)
	}

	if strings.TrimSpace(sessionId) == "" {
		return nil, fmt.Errorf("%w: empty sessionId", db.ErrDBQuery)
	}

	var message []db.EntryDB

	err := store.db.WithContext(ctx).Where("session_id = ?",
		sessionId).
		Find(&message).
		Error

	if err != nil {
		return nil, fmt.Errorf("%w: %v", db.ErrDBQuery, err)
	}

	if len(message) == 0 {
		return nil, fmt.Errorf("%w: session_id=%s", db.ErrRecordNotFound, sessionId)
	}

	result := make([]entities.Entry, 0, len(message))
	for _, entryDB := range message {
		result = append(result, db.EntryDbToEntry(entryDB))
	}

	return result, nil
}

// PutAnomaly Add new record to table
func (store *Store) PutAnomaly(ctx context.Context, msg entities.Entry) error {

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("%w: context error before query", db.ErrDBQuery)
	}

	if strings.TrimSpace(msg.SessionId) == "" {
		return fmt.Errorf("%w: empty sessionId", db.ErrDBQuery)
	}

	if math.IsNaN(msg.Frequency) || math.IsInf(msg.Frequency, 0) {
		return fmt.Errorf("%w: invalid frequency", db.ErrDBQuery)
	}

	if msg.Timestamp.IsZero() {
		return fmt.Errorf("%w: empty timestamp", db.ErrDBQuery)
	}

	result := store.db.WithContext(ctx).Create(db.EntryToEntryDB(msg))
	if result.Error != nil {
		if strings.Contains(result.Error.Error(), "duplicate key") {
			return db.ErrDuplicateEntry
		}
		return fmt.Errorf("%w: %v", db.ErrDBQuery, result.Error)
	}

	return nil
}

// DeleteAnomaly Delete anomaly from table
func (store *Store) DeleteAnomaly(ctx context.Context, sessionId string) error {
	result := store.db.WithContext(ctx).Delete(&db.EntryDB{}, "session_id = ?", sessionId)
	if result.Error != nil {
		return fmt.Errorf("%w: %v", db.ErrDBQuery, result.Error)
	}
	if result.RowsAffected == 0 {
		return db.ErrRecordNotFound
	}
	return nil
}
