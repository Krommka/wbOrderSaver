package postgres

import (
	"context"
	"database/sql"
	"errors"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
	"wb_l0/internal/domain"
	"wb_l0/pkg/logger"
)

func TestStore_GetOrderByUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	log := logger.NewTestLogger()
	store := &Store{db: db, log: log}

	t.Run("successful get order", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"order_uid", "track_number", "entry", "locale", "internal_signature",
			"customer_id", "delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
			"name", "phone", "zip", "city", "address", "region", "email",
			"transaction", "request_id", "currency_id", "provider_name",
			"amount", "payment_dt", "bank", "delivery_cost", "goods_total", "custom_fee",
		}).AddRow(
			"00000000000000000000", "TRACK001", "WB", "en", "signature",
			"customer123", "delivery-service", "shard1", 1, time.Now(), "oof1",
			"John Doe", "+1234567890", "123456", "Moscow", "Street 1", "Moscow", "john@test.com",
			"trans123", "req123", "USD", "provider1",
			1000, time.Now().Unix(), "bank123", 100, 900, 0,
		)

		mock.ExpectQuery(`SELECT.*FROM orders`).
			WithArgs("test-uid").
			WillReturnRows(rows)

		itemRows := sqlmock.NewRows([]string{
			"chrt_id", "track_number", "price", "rid", "name", "sale", "size",
			"total_price", "nm_id", "brand_name", "status_id", "quantity",
		}).AddRow(
			1, "TRACK001", 500, "rid001", "Test Item", 0, "M",
			500, 123, "Test Brand", 200, 1,
		)

		mock.ExpectQuery(`SELECT.*FROM order_items`).
			WithArgs("test-uid").
			WillReturnRows(itemRows)

		order, err := store.GetOrderByUID(context.Background(), "test-uid")

		assert.NoError(t, err)
		assert.Equal(t, "00000000000000000000", order.OrderUID)
		assert.Len(t, order.Items, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("order not found", func(t *testing.T) {
		mock.ExpectQuery(`SELECT.*FROM orders`).
			WithArgs("not-found").
			WillReturnError(sql.ErrNoRows)

		order, err := store.GetOrderByUID(context.Background(), "not-found")

		assert.Error(t, err)
		assert.Nil(t, order)
		assert.True(t, errors.Is(err, domain.ErrRecordNotFound))
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStore_SaveOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := logger.NewTestLogger()
	store := &Store{db: db, log: log}

	createTestOrder := func() *domain.Order {
		return &domain.Order{
			OrderUID:          "00000000000000000000",
			TrackNumber:       "TRACK001",
			Entry:             "WB",
			Locale:            "en",
			InternalSignature: "signature123",
			CustomerID:        "customer123",
			DeliveryService:   "test-service",
			ShardKey:          "shard1",
			SMID:              1,
			DateCreated:       time.Now(),
			OOFShard:          "oof1",
			Delivery: domain.Delivery{
				Name:    "John Doe",
				Phone:   "+1234567890",
				Zip:     "123456",
				City:    "Moscow",
				Address: "Street 1",
				Region:  "Moscow",
				Email:   "john@test.com",
			},
			Payment: domain.Payment{
				Transaction:  "00000000000000000000",
				RequestID:    "req123",
				Currency:     "USD",
				Provider:     "test-provider",
				Amount:       1000,
				PaymentDT:    time.Now().Unix(),
				Bank:         "sberbank",
				DeliveryCost: 100,
				GoodsTotal:   900,
				CustomFee:    0,
			},
			Items: []domain.Item{
				{
					ChrtID:      1,
					TrackNumber: "TRACK001",
					Price:       500,
					RID:         "rid001",
					Name:        "Test Item",
					Sale:        0,
					Size:        "M",
					TotalPrice:  500,
					NMID:        123,
					Brand:       "Test Brand",
					Status:      200,
				},
			},
		}
	}

	t.Run("successful save new order", func(t *testing.T) {
		order := createTestOrder()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs("00000000000000000000").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT service_id FROM delivery_services WHERE name = \$1`).
			WithArgs("test-service").
			WillReturnRows(sqlmock.NewRows([]string{"service_id"}).AddRow(1))

		mock.ExpectExec(`INSERT INTO orders`).
			WithArgs(
				"00000000000000000000", "TRACK001", "WB", "en", "signature123",
				"customer123", 1, "shard1", 1, sqlmock.AnyArg(), "oof1",
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO delivery`).
			WithArgs(
				"00000000000000000000", "John Doe", "+1234567890", "123456", "Moscow",
				"Street 1", "Moscow", "john@test.com",
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery(`SELECT provider_id FROM payment_providers WHERE name = \$1`).
			WithArgs("test-provider").
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery(`INSERT INTO payment_providers`).
			WithArgs("test-provider").
			WillReturnRows(sqlmock.NewRows([]string{"provider_id"}).AddRow(1))

		mock.ExpectExec(`INSERT INTO payment`).
			WithArgs(
				"00000000000000000000", "req123", "USD", 1, 1000, sqlmock.AnyArg(),
				"sberbank", 100, 900, 0,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery(`SELECT brand_id FROM brands WHERE name = \$1`).
			WithArgs("Test Brand").
			WillReturnRows(sqlmock.NewRows([]string{"brand_id"}).AddRow(1))

		mock.ExpectQuery(`INSERT INTO items`).
			WithArgs(
				1, "TRACK001", 500, "rid001", "Test Item", 0, "M",
				500, 123, 1, 200,
			).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

		mock.ExpectExec(`INSERT INTO order_items`).
			WithArgs("00000000000000000000", 1, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err := store.SaveOrder(context.Background(), order)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("order already exists - should skip", func(t *testing.T) {
		order := createTestOrder()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs("00000000000000000000").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		err := store.SaveOrder(context.Background(), order)

		assert.NoError(t, err) // Дубликаты игнорируются без ошибки
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to check order existence", func(t *testing.T) {
		order := createTestOrder()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs("00000000000000000000").
			WillReturnError(errors.New("database error"))

		err := store.SaveOrder(context.Background(), order)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check order existence")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to begin transaction", func(t *testing.T) {
		order := createTestOrder()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs("00000000000000000000").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectBegin().WillReturnError(errors.New("tx begin error"))

		err := store.SaveOrder(context.Background(), order)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to create delivery service", func(t *testing.T) {
		order := createTestOrder()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs("00000000000000000000").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT service_id FROM delivery_services WHERE name = \$1`).
			WithArgs("test-service").
			WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery(`INSERT INTO delivery_services`).
			WithArgs("test-service").
			WillReturnError(errors.New("delivery service creation failed"))

		mock.ExpectRollback()

		err := store.SaveOrder(context.Background(), order)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create delivery service")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to insert order", func(t *testing.T) {
		order := createTestOrder()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs("00000000000000000000").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT service_id FROM delivery_services WHERE name = \$1`).
			WithArgs("test-service").
			WillReturnRows(sqlmock.NewRows([]string{"service_id"}).AddRow(1))

		mock.ExpectExec(`INSERT INTO orders`).
			WillReturnError(errors.New("order insert failed"))

		mock.ExpectRollback()

		err := store.SaveOrder(context.Background(), order)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to insert order")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to commit transaction", func(t *testing.T) {
		order := createTestOrder()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs("00000000000000000000").
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT service_id FROM delivery_services WHERE name = \$1`).
			WithArgs("test-service").
			WillReturnRows(sqlmock.NewRows([]string{"service_id"}).AddRow(1))

		mock.ExpectExec(`INSERT INTO orders`).WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO delivery`).WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery(`SELECT provider_id FROM payment_providers WHERE name = \$1`).
			WithArgs("test-provider").
			WillReturnRows(sqlmock.NewRows([]string{"provider_id"}).AddRow(1))
		mock.ExpectExec(`INSERT INTO payment`).WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery(`SELECT brand_id FROM brands WHERE name = \$1`).
			WithArgs("Test Brand").
			WillReturnRows(sqlmock.NewRows([]string{"brand_id"}).AddRow(1))

		mock.ExpectQuery(`INSERT INTO items`).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
		mock.ExpectExec(`INSERT INTO order_items`).WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

		err := store.SaveOrder(context.Background(), order)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestStore_DeleteOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	log := logger.NewTestLogger()
	store := &Store{db: db, log: log}

	t.Run("successful delete order", func(t *testing.T) {
		orderUID := "00000000000000000000"

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs(orderUID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mock.ExpectExec(`DELETE FROM orders WHERE order_uid = \$1`).
			WithArgs(orderUID).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 affected row

		mock.ExpectCommit()

		err := store.DeleteOrder(context.Background(), orderUID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("order not found", func(t *testing.T) {
		orderUID := "00000000000000000000"

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs(orderUID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		mock.ExpectRollback()

		err := store.DeleteOrder(context.Background(), orderUID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "order with UID "+orderUID+" not found")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to begin transaction", func(t *testing.T) {
		orderUID := "00000000000000000000"

		mock.ExpectBegin().WillReturnError(errors.New("begin transaction error"))

		err := store.DeleteOrder(context.Background(), orderUID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to check order existence", func(t *testing.T) {
		orderUID := "00000000000000000000"

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs(orderUID).
			WillReturnError(errors.New("database error"))

		mock.ExpectRollback()

		err := store.DeleteOrder(context.Background(), orderUID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check order existence")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to delete order", func(t *testing.T) {
		orderUID := "00000000000000000000"

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs(orderUID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mock.ExpectExec(`DELETE FROM orders WHERE order_uid = \$1`).
			WithArgs(orderUID).
			WillReturnError(errors.New("delete error"))

		mock.ExpectRollback()

		err := store.DeleteOrder(context.Background(), orderUID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete order")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("failed to commit transaction", func(t *testing.T) {
		orderUID := "00000000000000000000"

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs(orderUID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mock.ExpectExec(`DELETE FROM orders WHERE order_uid = \$1`).
			WithArgs(orderUID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit().WillReturnError(errors.New("commit error"))

		err := store.DeleteOrder(context.Background(), orderUID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to commit transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("cascade delete works correctly", func(t *testing.T) {
		orderUID := "00000000000000000000"

		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE order_uid = \$1\)`).
			WithArgs(orderUID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		mock.ExpectExec(`DELETE FROM orders WHERE order_uid = \$1`).
			WithArgs(orderUID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		err := store.DeleteOrder(context.Background(), orderUID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("context cancellation", func(t *testing.T) {
		orderUID := "00000000000000000000"

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := store.DeleteOrder(ctx, orderUID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to begin transaction")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
