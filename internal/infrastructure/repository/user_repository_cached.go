package repository

import (
	"context"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/infrastructure/cache"
	"time"
)

const userCacheTTL = 10 * time.Minute

type CachedUserRepository struct {
	repo  user.Repository
	cache *cache.Cache[int64, *user.User]
}

func NewCachedUserRepository(repo user.Repository) *CachedUserRepository {
	return &CachedUserRepository{
		repo:  repo,
		cache: cache.New[int64, *user.User](userCacheTTL),
	}
}

func (r *CachedUserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	return r.repo.FindAll(ctx)
}

func (r *CachedUserRepository) FindByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	if cached, ok := r.cache.Get(telegramID); ok {
		return cached, nil
	}

	u, err := r.repo.FindByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	r.cache.Set(telegramID, u)

	return u, nil
}
