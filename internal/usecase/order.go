package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"wb_l0/internal/domain"
)

type OrderUsecase struct {
	store      store
	retryCount int
	log        *slog.Logger
}

func NewOrderUsecase(store store, retryCount int, log *slog.Logger) *OrderUsecase {
	return &OrderUsecase{store: store, retryCount: retryCount, log: log}
}

func (uc *OrderUsecase) GetOrder(ctx context.Context, orderUID string) (*domain.Order, error) {
	order, err := uc.store.GetOrderByUID(ctx, orderUID)
	if err != nil {
		return nil, err
	}
	return order, nil
}
func (uc *OrderUsecase) CreateOrder(ctx context.Context, order domain.Order) error {
	startTime := time.Now()
	uc.log.Info("Order creation started",
		"order_uid", order.OrderUID,
		"customer_id", order.CustomerID,
		"total_amount", order.Payment.Amount,
	)

	if err := uc.validateOrder(order); err != nil {
		return err
	}

	uc.log.Debug("Business validation passed",
		"order_uid", order.OrderUID,
	)

	var lastErr error

	for i := 0; i < uc.retryCount; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
			err := uc.store.SaveOrder(ctx, &order)
			if err == nil {

				uc.log.Info("Order business processing completed",
					"order_uid", order.OrderUID,
					"items_count", len(order.Items),
					"processing_time_ms", time.Since(startTime).Milliseconds(),
				)
				return nil
			}

			lastErr = err
			uc.log.Error("Retry for order %s failed: %v",
				"error", err,
				"retry", i+1,
				"retry_count", uc.retryCount,
				"order_uid", order.OrderUID,
			)

			delay := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(delay)
		}
	}

	uc.log.Error("Business processing failed",
		"order_uid", order.OrderUID,
		"error", lastErr,
		"error_type", "business",
	)
	return lastErr
}

func (uc *OrderUsecase) validateOrder(order domain.Order) error {
	if err := order.Validate(); err != nil {
		return err
	}
	return nil
}
