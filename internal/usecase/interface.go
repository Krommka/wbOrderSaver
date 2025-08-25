package usecase

import (
	"context"
	"wb_l0/internal/domain"
)

type store interface {
	SaveOrder(ctx context.Context, order *domain.Order) error
	GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error)
	DeleteOrder(ctx context.Context, orderUID string) error
}
