package postgres

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"Go_Team00.ID_376234-Team_TL_barievel/configs"
	"Go_Team00.ID_376234-Team_TL_barievel/db"
	"Go_Team00.ID_376234-Team_TL_barievel/internal/entities"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var TestDBConfig = configs.Config{
	DB: configs.DBConfig{
		Host:           "localhost",
		Port:           "5400",
		User:           "postgres",
		Password:       "postgres",
		Name:           "anomalyDetection",
		ConnectTimeout: "1",
		Retries:        1,
	},
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		TestDBConfig.DB.Host,
		TestDBConfig.DB.Port,
		TestDBConfig.DB.User,
		TestDBConfig.DB.Password,
		TestDBConfig.DB.Name,
	)

	testDb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	return testDb
}

func cleanupTestDB(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.Exec("TRUNCATE TABLE entry_dbs RESTART IDENTITY CASCADE").Error; err != nil {
		t.Errorf("Failed to clean test database: %v", err)
	}
}

func TestNewStore(t *testing.T) {
	ctx := context.Background()
	t.Run("successful creation", func(t *testing.T) {
		testDB := setupTestDB(t)
		defer cleanupTestDB(t, testDB)

		store, err := NewStore(ctx, TestDBConfig)
		defer func() {
			if store != nil {
				store.Disconnect(ctx)
			}
		}()
		require.NoError(t, err, "NewStore() should not return error")
		assert.NotNil(t, store, "NewStore() should return valid store instance")

	})
	t.Run("connection error", func(t *testing.T) {
		timeoutCtx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		incorrectConfig := TestDBConfig
		incorrectConfig.DB.Host = "invalid_host"

		store, err := NewStore(timeoutCtx, incorrectConfig)

		assert.Error(t, err, "Expected error for invalid connection, got nil")
		assert.Nil(t, store, "NewStore() should return a nil store")
	})
}

func TestStore_Connect(t *testing.T) {
	ctx := context.Background()
	t.Run("successful connection", func(t *testing.T) {
		store := &Store{}
		err := store.Connect(ctx, TestDBConfig)
		defer store.Disconnect(ctx)
		require.NoError(t, err, "Connect() should not return error")
		require.NotNil(t, store, "Connect() should return valid store instance")
	})

	t.Run("connection timeout", func(t *testing.T) {
		incorrectConfig := TestDBConfig
		incorrectConfig.DB.Host = "invalid_host"
		store := &Store{}
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		err := store.Connect(timeoutCtx, incorrectConfig)
		assert.Error(t, err, "Expected timeout error, got nil")
		assert.ErrorIs(t, err, db.ErrTimeout)
	})

	t.Run("canceled context", func(t *testing.T) {
		incorrectConfig := TestDBConfig
		incorrectConfig.DB.Host = "invalid_host"
		store := &Store{}
		timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
		defer cancel()
		err := store.Connect(timeoutCtx, incorrectConfig)
		assert.Error(t, err, "Expected timeout error, got nil")
		assert.Contains(t, err.Error(), "cancelled before connection",
			"Error should contain specific text")
	})
}

func TestStore_GetAnomaly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Подготовка тестовых данных
	testSession1 := entities.Entry{
		SessionId: "test_session1",
		Frequency: 123.45,
		Timestamp: time.Now().Truncate(time.Millisecond),
	}
	time.Sleep(10 * time.Millisecond)
	testSession1second := entities.Entry{
		SessionId: "test_session1",
		Frequency: 678.90,
		Timestamp: time.Now().Truncate(time.Millisecond),
	}
	time.Sleep(10 * time.Millisecond)
	testSession2 := entities.Entry{
		SessionId: "test_session2",
		Frequency: 123.45,
		Timestamp: time.Now().Truncate(time.Millisecond),
	}

	responseSlice1 := []entities.Entry{testSession1, testSession1second}
	responseSlice2 := []entities.Entry{testSession2}

	if err := testDB.Create(db.EntryToEntryDB(testSession1)).Error; err != nil {
		t.Fatalf("Failed to create test anomaly: %v", err)
	}
	if err := testDB.Create(db.EntryToEntryDB(testSession1second)).Error; err != nil {
		t.Fatalf("Failed to create test anomaly: %v", err)
	}
	if err := testDB.Create(db.EntryToEntryDB(testSession2)).Error; err != nil {
		t.Fatalf("Failed to create test anomaly: %v", err)
	}

	t.Run("Test multiple anomaly", func(t *testing.T) {
		store := &Store{db: testDB}
		got, err := store.GetAnomaly(ctx, "test_session1")

		if err != nil {
			t.Errorf("GetAnomaly() error = %v", err)
			return
		}

		if len(got) != len(responseSlice1) {
			t.Errorf("GetAnomaly() returned %d entries, want %d", len(got), len(responseSlice1))
		}

		for i, v := range got {
			if !reflect.DeepEqual(v, responseSlice1[i]) {
				t.Errorf("GetAnomaly() returned %v, want %v", v, responseSlice1[i])
			}
		}
	})

	t.Run("Test single anomaly", func(t *testing.T) {
		store := &Store{db: testDB}
		got, err := store.GetAnomaly(ctx, "test_session2")
		if err != nil {
			t.Errorf("GetAnomaly() error = %v", err)
			return
		}
		if len(got) != len(responseSlice2) {
			t.Errorf("GetAnomaly() returned %d entries, want %d", len(got), len(responseSlice2))
		}
		if !reflect.DeepEqual(testSession2, responseSlice2[0]) {
			t.Errorf("GetAnomaly() returned %v, want %v", got[0], responseSlice2[0])
		}
	})

	t.Run("Test not exist anomaly", func(t *testing.T) {
		store := &Store{db: testDB}
		got, err := store.GetAnomaly(ctx, "test_session0")
		if err == nil && !errors.Is(err, db.ErrRecordNotFound) {
			t.Errorf("GetAnomaly() want error, got nil. Response was %v", got)
		}
	})
}

func TestStore_PutAnomaly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Подготовка тестовых данных
	testAnomaly := entities.Entry{
		SessionId: "test_session",
		Frequency: 123.45,
		Timestamp: time.Now().Truncate(time.Millisecond),
	}
	time.Sleep(10 * time.Millisecond)
	testAnomalySameSession := entities.Entry{
		SessionId: "test_session",
		Frequency: 123.45,
		Timestamp: time.Now().Truncate(time.Millisecond),
	}
	testAnomalyIncorrectId := entities.Entry{
		SessionId: "",
		Frequency: 123.45,
		Timestamp: time.Now().Truncate(time.Millisecond),
	}
	testAnomalyIncorrectFrequency := entities.Entry{
		SessionId: "test_session",
		Frequency: math.NaN(),
		Timestamp: time.Now().Truncate(time.Millisecond),
	}
	testAnomalyIncorrectTimestamp := entities.Entry{
		SessionId: "test_session",
		Frequency: 456.78,
		Timestamp: time.Time{},
	}

	testTable := []struct {
		name        string
		anomaly     entities.Entry
		expectedErr error
	}{
		{
			name:        "non-existent anomaly",
			anomaly:     testAnomaly,
			expectedErr: nil,
		},
		{
			name:        "same session anomaly",
			anomaly:     testAnomalySameSession,
			expectedErr: nil,
		},
		{
			name:        "empty id",
			anomaly:     testAnomalyIncorrectId,
			expectedErr: db.ErrDBQuery,
		},
		{
			name:        "empty frequency",
			anomaly:     testAnomalyIncorrectFrequency,
			expectedErr: db.ErrDBQuery,
		},
		{
			name:        "empty timestamp",
			anomaly:     testAnomalyIncorrectTimestamp,
			expectedErr: db.ErrDBQuery,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			store := &Store{db: testDB}
			err := store.PutAnomaly(ctx, testCase.anomaly)

			if !errors.Is(err, testCase.expectedErr) {
				t.Errorf("PutAnomaly() error = %v, expectedDuplicateErr %v", err, testCase.expectedErr)
				return
			}
		})
	}
}

func TestStore_DeleteAnomaly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)

	// Подготовка тестовых данных
	testAnomaly := entities.Entry{
		SessionId: "test_session",
		Frequency: 123.45,
		Timestamp: time.Now().Truncate(time.Millisecond),
	}
	if err := testDB.Create(db.EntryToEntryDB(testAnomaly)).Error; err != nil {
		t.Fatalf("Failed to create test anomaly: %v", err)
	}

	testTable := []struct {
		name        string
		sessionId   string
		want        entities.Entry
		expectedErr error
	}{
		{
			name:        "existing record",
			sessionId:   "test_session",
			expectedErr: nil,
		},
		{
			name:        "non-existent record",
			sessionId:   "nonexistent",
			expectedErr: db.ErrRecordNotFound,
		},
		{
			name:        "empty session id",
			sessionId:   "",
			expectedErr: db.ErrRecordNotFound,
		},
	}

	for _, testCase := range testTable {
		t.Run(testCase.name, func(t *testing.T) {
			store := &Store{db: testDB}
			err := store.DeleteAnomaly(ctx, testCase.sessionId)

			if !errors.Is(err, testCase.expectedErr) {
				t.Errorf("DeleteAnomaly() error = %v, expectedErr %v", err, testCase.expectedErr)
				return
			}

		})
	}
}

func TestStore_Disconnect(t *testing.T) {
	mockDb, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create mock DB")
	defer mockDb.Close()

	gormDb, err := gorm.Open(postgres.New(postgres.Config{
		Conn: mockDb,
	}), &gorm.Config{})
	require.NoError(t, err, "Failed to create mockDB")

	t.Run("success disconnect", func(t *testing.T) {
		ctx := context.Background()
		store := &Store{db: gormDb}
		mock.ExpectClose()
		err = store.Disconnect(ctx)

		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty store", func(t *testing.T) {

		ctx := context.Background()
		store := &Store{}
		err = store.Disconnect(ctx)

		require.Contains(t, err.Error(), "store is nil", "Error should contain store nil")

	})

	t.Run("context timeout", func(t *testing.T) {

		// Создаем Store
		store := &Store{db: gormDb}

		// Создаем контекст с таймаутом
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()

		// Вызываем метод Disconnect
		err = store.Disconnect(ctx)

		// Проверяем, что произошел таймаут
		require.Error(t, err)
		require.Contains(t, err.Error(), "database shutdown timed out", "Error should contain timed out text")

		// Проверяем выполнение ожиданий
		err = mock.ExpectationsWereMet()
		require.NoError(t, err)

	})

}

func TestMigrate(t *testing.T) {
	testDB := setupTestDB(t)
	defer cleanupTestDB(t, testDB)
	t.Run("successful migration", func(t *testing.T) {
		store := &Store{db: testDB}
		err := store.Migrate()
		require.NoError(t, err, "Migrate() should not return error")

	})
	t.Run("failed migration", func(t *testing.T) {

		mockDb, _, err := sqlmock.New()
		require.NoError(t, err, "Failed to create mock DB")
		defer mockDb.Close()

		brokenDB, err := gorm.Open(postgres.New(postgres.Config{
			Conn: mockDb, // Отсутствует подключение
		}), &gorm.Config{})
		require.NoError(t, err, "Failed to create broken GORM DB")

		store := &Store{db: brokenDB}

		// Вызываем Migrate и ожидаем ошибку
		err = store.Migrate()
		require.Error(t, err, "Migrate() should return error on broken DB")
		require.Contains(t, err.Error(), "error migrating database", "Error message should indicate migration issue")

	})
}
