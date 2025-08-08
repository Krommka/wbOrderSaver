package redisCache

import (
	"KinopoiskTwoActors/configs"
	"KinopoiskTwoActors/internal/domain"
	"context"
	"encoding/json"
	"errors"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"strconv"
	"time"
)

type RedisRepo struct {
	client *redis.Client
	prefix string
	log    *slog.Logger
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

	if err := db.Ping(ctx).Err(); err != nil {
		log.Error("Ошибка подключения к Redis", "error", err)
		return &RedisRepo{}, err
	}
	log.Info("Клиент подключен к Redis", "host", cfg.RD.Host)

	return &RedisRepo{
		client: db,
		prefix: prefix,
		log:    log,
	}, nil
}

func (r *RedisRepo) GetMovieByID(ctx context.Context, movieID int) (domain.Movie, error) {
	r.log.Debug("Получение фильма в Redis", "movieID", movieID)
	key := r.prefix + strconv.FormatInt(int64(movieID), 10)
	data, err := r.client.Get(ctx, key).Bytes()
	if errors.Is(err, redis.Nil) {
		r.log.Debug("Фильм в базе Redis не найден", "movieID", movieID)
		return domain.Movie{}, domain.ErrRecordNotFound
	} else if err != nil {
		r.log.Debug("Ошибка получение данных из Redis", "movieID", movieID)
		return domain.Movie{}, err
	}

	var movie domain.Movie
	if err := json.Unmarshal(data, &movie); err != nil {
		r.log.Debug("Ошибка конвертации из базы Redis", "movieID", movieID)
		return domain.Movie{}, err
	}
	return movie, nil
}

func (r *RedisRepo) SetMovie(ctx context.Context, movie domain.Movie) error {
	key := r.prefix + strconv.FormatInt(int64(movie.ID), 10)
	data, err := json.Marshal(movie)
	if err != nil {
		r.log.Error("Отправка фильма в базу Redis", "error", err)
		return err
	}
	return r.client.Set(ctx, key, data, 24*time.Hour).Err()
}
