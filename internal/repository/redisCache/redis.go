package redisCache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"
	"wb_l0/configs"
	"wb_l0/internal/domain"

	"github.com/redis/go-redis/v9"
)

type RedisRepo struct {
	client   *redis.Client
	prefix   string
	capacity int
	log      *slog.Logger
}

func NewCache(ctx context.Context, cfg *configs.Config, prefix string, log *slog.Logger) (*RedisRepo,
	error) {
	db := redis.NewClient(&redis.Options{
		Addr:         cfg.RD.Host,
		DB:           cfg.RD.DB,
		Password:     cfg.RD.Password,
		MaxRetries:   cfg.RD.MaxRetries,
		DialTimeout:  cfg.RD.DialTimeout,
		ReadTimeout:  cfg.RD.ReadTimeout,
		WriteTimeout: cfg.RD.WriteTimeout,
	})

	log.Info("attempting to connect to Redis", "host", cfg.RD.Host, "db", cfg.RD.DB)

	if err := db.Ping(ctx).Err(); err != nil {
		log.Error("Redis connection failed", "error", err, "host", cfg.RD.Host)
		return &RedisRepo{}, err
	}
	log.Info("successfully connected to Redis", "host", cfg.RD.Host)

	return &RedisRepo{
		client:   db,
		prefix:   prefix,
		capacity: cfg.RD.Capacity,
		log:      log,
	}, nil
}

func (r *RedisRepo) GetOrderByUID(ctx context.Context, orderUID string) (*domain.Order, error) {
	order := &domain.Order{}
	r.log.Debug("Getting order from Redis", "orderUID", orderUID)
	key := r.prefix + orderUID
	data, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		r.log.Debug("Order not found", "orderUID", orderUID)
		return order, domain.ErrRecordNotFound
	} else if err != nil {
		r.log.Debug("error getting from redis", "orderUID", orderUID)
		return order, err
	}

	if err := json.Unmarshal(data, &order); err != nil {
		r.log.Debug("error converting from redis", "orderUID", orderUID)
		return order, err
	}
	return order, nil
}

func (r *RedisRepo) SaveOrder(ctx context.Context, order *domain.Order) error {
	r.log.Debug("starting to set order in cache")

	key := r.prefix + order.OrderUID
	data, err := json.Marshal(order)
	if err != nil {
		r.log.Error("error while setting to Redis", "error", err, "orderUID", order.OrderUID)
		return err
	}
	err = r.client.Set(ctx, key, data, 0).Err()
	if err != nil {
		return err
	}
	r.log.Debug("order data stored in Redis", "orderUID", order.OrderUID)

	timestamp := float64(time.Now().UnixNano())
	sortedSetKey := r.prefix + "recent_orders"
	err = r.client.ZAdd(ctx, sortedSetKey, redis.Z{
		Score:  timestamp,
		Member: order.OrderUID,
	}).Err()
	if err != nil {
		return err
	}
	r.log.Debug("order added to recent_orders sorted set", "timestamp", timestamp)

	// Удаляем старые заказы если превышен лимит
	if r.capacity > 0 {
		count, err := r.client.ZCard(ctx, sortedSetKey).Result()
		if err != nil {
			return err
		}

		if count > int64(r.capacity) {
			// Получаем UID заказов, которые нужно удалить
			uidsToRemove, err := r.client.ZRange(ctx, sortedSetKey, 0, count-int64(r.capacity)-1).Result()
			if err != nil {
				r.log.Error("failed to get old orders for removal", "error", err)
				return err
			}

			// Удаляем из sorted set
			removedFromSet, err := r.client.ZRemRangeByRank(ctx, sortedSetKey, 0, count-int64(r.capacity)-1).Result()
			if err != nil {
				r.log.Error("failed to remove from sorted set", "error", err)
				return err
			}

			// Удаляем сами данные заказов
			if len(uidsToRemove) > 0 {
				keysToDelete := make([]string, len(uidsToRemove))
				for i, uid := range uidsToRemove {
					keysToDelete[i] = r.prefix + uid
				}

				deleted, err := r.client.Del(ctx, keysToDelete...).Result()
				if err != nil {
					r.log.Error("failed to delete order data", "error", err)
				} else {
					r.log.Debug("deleted old orders",
						"from_set", removedFromSet,
						"from_data", deleted,
						"uids", uidsToRemove)
				}
			}
		}
	}

	r.log.Info("order successfully cached", "orderUID", order.OrderUID)
	return nil
}

func (r *RedisRepo) CountOrders(ctx context.Context) (int, error) {
	res, err := r.client.ZCard(ctx, r.prefix+"recent_orders").Result()
	return int(res), err
}
