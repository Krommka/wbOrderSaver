package cachedRepo

import (
	"context"
	"log/slog"
	"time"
	"wb_l0/configs"
	"wb_l0/internal/domain"
)

type OrderRepository interface {
	GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error)
	DeleteOrder(ctx context.Context, orderUID string) error
	SaveOrder(ctx context.Context, order *domain.Order) error
	GetLastOrdersUIDs(ctx context.Context, limit int) ([]string, error)
}

type CacheRepository interface {
	GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error)
	SaveOrder(ctx context.Context, order *domain.Order) error
	CountOrders(ctx context.Context) (int, error)
}

type CachedRepo struct {
	repo  OrderRepository
	cache CacheRepository
	log   *slog.Logger
	cfg   *configs.Config
}

func NewCachedRepo(ctx context.Context, repo OrderRepository, cache CacheRepository, log *slog.Logger, cfg *configs.Config) *CachedRepo {
	log.Info("initializing cached repo", "warmUp", cfg.RD.WarmUp, "capacity", cfg.RD.Capacity)
	if cfg.RD.WarmUp == true {
		count, err := cache.CountOrders(ctx)
		log.Debug("cache repo count", "count", count)
		if err != nil {
			log.Warn("failed to count orders from database", "error", err)
		}
		if count < cfg.RD.Capacity {
			log.Info("starting cache repo warmUp", "count", count, "capacity", cfg.RD.Capacity)
			go func() {
				err = warmUpCache(ctx, cfg.RD.Capacity, repo, cache, log)
				if err != nil {
					log.Warn("failed to warmUpCache", "error", err)
				}
			}()
		}
	}
	log.Info("cache initialized")
	return &CachedRepo{
		repo:  repo,
		cache: cache,
		log:   log,
		cfg:   cfg,
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

func warmUpCache(ctx context.Context, capacity int, repo OrderRepository, cache CacheRepository, log *slog.Logger) error {

	log.Info("Starting cache warm-up process")
	startTime := time.Now()
	limit := capacity
	orderUIDs, err := repo.GetLastOrdersUIDs(ctx, limit)
	if err != nil {
		log.Error("Failed to get last orders UIDs for cache warm-up",
			"error", err.Error())
		return err
	}

	log.Info("Retrieved orders for cache warm-up",
		"count", len(orderUIDs))

	successCount := 0
	for i, orderUID := range orderUIDs {
		select {
		case <-ctx.Done():
			log.Warn("Cache warm-up interrupted by context cancellation")
			return ctx.Err()
		default:
			order, err := repo.GetOrderByUID(ctx, orderUID)
			if err != nil {
				log.Warn("Failed to get order for cache warm-up",
					"order_uid", orderUID,
					"error", err.Error(),
					"index", i)
				continue
			}

			err = cache.SaveOrder(ctx, order)
			if err != nil {
				log.Warn("Failed to save order to cache during warm-up",
					"order_uid", orderUID,
					"error", err.Error(),
					"index", i)
				continue
			}

			successCount++

			if (i+1)%10 == 0 {
				log.Info("Cache warm-up progress",
					"processed", i+1,
					"total", len(orderUIDs),
					"success", successCount)
			}
		}
	}

	log.Info("Cache warm-up completed",
		"total_orders", len(orderUIDs),
		"successful", successCount,
		"duration_ms", time.Since(startTime).Milliseconds())
	return nil
}
