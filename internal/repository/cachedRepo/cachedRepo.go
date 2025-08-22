package cachedRepo

import (
	"context"
	"log/slog"
	"wb_l0/internal/domain"
)

type OrderRepository interface {
	GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error)
	DeleteOrder(ctx context.Context, orderUID string) error
	SaveOrder(ctx context.Context, order *domain.Order) error
}

type CacheRepository interface {
	GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error)
	SaveOrder(ctx context.Context, order *domain.Order) error
}

type CachedRepo struct {
	repo  OrderRepository
	cache CacheRepository
	log   *slog.Logger
}

func NewCachedRepo(repo OrderRepository, cache CacheRepository, log *slog.Logger) *CachedRepo {

	return &CachedRepo{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}

func (r *CachedRepo) DeleteOrder(ctx context.Context, orderUID string) error {
	return r.repo.DeleteOrder(ctx, orderUID)
}

func (r *CachedRepo) GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error) {
	order, err := r.cache.GetOrderByUID(ctx, orderUID)
	r.log.Debug("attempting to get order from cache", "orderUID", orderUID)
	if err == nil && order != nil {
		r.log.Debug("order found in cache")
		return order, nil
	}
	if err != nil && err != domain.ErrRecordNotFound {
		r.log.Warn("error getting from cache, falling back to database", "error", err, "orderUID", orderUID)
	}
	r.log.Debug("order not found in cache, querying database", "orderUID", orderUID)

	order, err = r.repo.GetOrderByUID(ctx, orderUID)
	if err != nil {
		r.log.Error("failed to get order from database", "error", err)
		return nil, err
	}
	r.log.Debug("order found in database, saving to cache")

}
