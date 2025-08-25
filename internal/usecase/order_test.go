package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"wb_l0/internal/domain"
	"wb_l0/internal/usecase"
	"wb_l0/pkg/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error) {
	args := m.Called(ctx, orderUID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Order), args.Error(1)
}

func (m *MockStore) DeleteOrder(ctx context.Context, orderUID string) error {
	args := m.Called(ctx, orderUID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Error(1)
}

func (m *MockStore) SaveOrder(ctx context.Context, order *domain.Order) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func TestOrderUsecase_GetOrder(t *testing.T) {
	log := logger.NewTestLogger()
	mockStore := new(MockStore)
	uc := usecase.NewOrderUsecase(mockStore, 3, log)

	t.Run("successful get order", func(t *testing.T) {
		expectedOrder := &domain.Order{
			OrderUID:   "test-uid",
			CustomerID: "test-customer",
		}

		mockStore.On("GetOrderByUID", mock.Anything, "test-uid").
			Return(expectedOrder, nil).
			Once()

		order, err := uc.GetOrder(context.Background(), "test-uid")

		assert.NoError(t, err)
		assert.Equal(t, expectedOrder, order)
		mockStore.AssertExpectations(t)
	})

	t.Run("order not found", func(t *testing.T) {
		mockStore.On("GetOrderByUID", mock.Anything, "not-found").
			Return(nil, domain.ErrRecordNotFound).
			Once()

		order, err := uc.GetOrder(context.Background(), "not-found")

		assert.Error(t, err)
		assert.Nil(t, order)
		assert.True(t, errors.Is(err, domain.ErrRecordNotFound))
		mockStore.AssertExpectations(t)
	})
}

func TestOrderUsecase_CreateOrder(t *testing.T) {
	log := logger.NewTestLogger()
	mockStore := new(MockStore)
	uc := usecase.NewOrderUsecase(mockStore, 3, log)

	validOrder := domain.CreateTestOrder(1)

	t.Run("successful order creation", func(t *testing.T) {
		mockStore.On("SaveOrder", mock.Anything, mock.AnythingOfType("*domain.Order")).
			Return(nil).
			Once()

		err := uc.CreateOrder(context.Background(), validOrder)

		assert.NoError(t, err)
		mockStore.AssertExpectations(t)
	})

	t.Run("order validation failed", func(t *testing.T) {
		invalidOrder := validOrder
		invalidOrder.OrderUID = ""

		err := uc.CreateOrder(context.Background(), invalidOrder)

		assert.Error(t, err)
		mockStore.AssertNotCalled(t, "SaveOrder")
	})

	t.Run("retry on save failure", func(t *testing.T) {
		mockStore.On("SaveOrder", mock.Anything, mock.AnythingOfType("*domain.Order")).
			Return(errors.New("database error")).
			Times(3)

		err := uc.CreateOrder(context.Background(), validOrder)

		assert.Error(t, err)
		mockStore.AssertExpectations(t)
	})
}

func TestOrderUsecase_ContextCancellation(t *testing.T) {
	log := logger.NewTestLogger()
	mockStore := new(MockStore)
	uc := usecase.NewOrderUsecase(mockStore, 3, log)

	validOrder := domain.CreateTestOrder(1)

	t.Run("context cancellation during retry", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		mockStore.On("SaveOrder", mock.Anything, mock.AnythingOfType("*domain.Order")).
			Return(errors.New("database error")).
			Maybe()

		err := uc.CreateOrder(ctx, validOrder)

		assert.Error(t, err)
	})
}
