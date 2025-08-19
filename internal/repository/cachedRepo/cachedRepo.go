package cachedRepo

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"wb_l0/internal/domain"
)

type ActorFilmRepository interface {
	SearchActors(ctx context.Context, query string) ([]domain.Actor, error)
	GetMoviesIDByActorID(ctx context.Context, actorID int) ([]int, error)
	GetMovieByID(ctx context.Context, movieID int) (domain.Movie, error)
}

type CacheRepository interface {
	GetMovieByID(ctx context.Context, movieID int) (domain.Movie, error)
	SetMovie(ctx context.Context, movie domain.Movie) error
}

type CachedRepo struct {
	repo  ActorFilmRepository
	cache CacheRepository
	log   *slog.Logger
}

func NewCachedRepo(repo ActorFilmRepository, cache CacheRepository, log *slog.Logger) *CachedRepo {

	return &CachedRepo{
		repo:  repo,
		cache: cache,
		log:   log,
	}
}
