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

func (r *CachedRepo) GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error) {
	order, err := r.cache.GetOrderByUID(ctx, orderUID)
	r.log.Debug("attempting to get order from cache", "orderUID", orderUID)
	if err == nil && order != nil {
		r.log.Debug("order found in cache")
		return order, nil
	}
	if err != nil && err != domain.ErrRecordNotFound {
		r.log.Warn("error getting from cache, falling back to database", "error",
			err, "orderUID", orderUID)
	}
	r.log.Debug("order not found in cache, querying database", "orderUID", orderUID)

	order, err = r.repo.GetOrderByUID(ctx, orderUID)
	if err != nil {
		r.log.Error("failed to get order from database", "error", err)
		return nil, err
	}
	r.log.Debug("order found in database, saving to cache")

	if err := r.cache.SaveOrder(ctx, order); err != nil {
		r.log.Warn("failed to save order to cache", "error", err)
	}

	r.log.Info("order retrieved successfully", "source", "database")
	return order, nil

}

func (r *CachedRepo) SaveOrder(ctx context.Context, order *domain.Order) error {

	r.log.Debug("saving order to database")

	if err := r.repo.SaveOrder(ctx, order); err != nil {
		r.log.Error("failed to save order to database", "error", err,
			"orderUID", order.OrderUID)
		return err
	}

	r.log.Debug("order saved to database, updating cache")

	if err := r.cache.SaveOrder(ctx, order); err != nil {
		r.log.Warn("failed to save order to cache", "error", err,
			"orderUID", order.OrderUID)
	}

	r.log.Info("order saved successfully", "orderUID", order.OrderUID)
	return nil
}

func (r *CachedRepo) DeleteOrder(ctx context.Context, orderUID string) error {
	r.log.Debug("deleting order from database")

	if err := r.repo.DeleteOrder(ctx, orderUID); err != nil {
		r.log.Error("failed to delete order from database", "error", err)
		return err
	}
	r.log.Info("order deleted from database successfully", "orderUID", orderUID)
	return nil
}
