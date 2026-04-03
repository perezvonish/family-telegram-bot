package repository

import (
	"context"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/infrastructure/cache"
	"strings"
	"time"
)

const userCacheTTL = 10 * time.Minute

type CachedUserRepository struct {
	repo          user.Repository
	telegramCache *cache.Cache[int64, *user.User]
	usernameCache *cache.Cache[string, *user.User]
}

func NewCachedUserRepository(repo user.Repository) *CachedUserRepository {
	return &CachedUserRepository{
		repo:          repo,
		telegramCache: cache.New[int64, *user.User](userCacheTTL),
		usernameCache: cache.New[string, *user.User](userCacheTTL),
	}
}

func (r *CachedUserRepository) FindAll(ctx context.Context) ([]*user.User, error) {
	return r.repo.FindAll(ctx)
}

func (r *CachedUserRepository) FindByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	if cached, ok := r.telegramCache.Get(telegramID); ok {
		return cached, nil
	}

	u, err := r.repo.FindByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	r.telegramCache.Set(telegramID, u)
	r.usernameCache.Set(normalizeUsernameKey(u.Username), u)

	return u, nil
}

func (r *CachedUserRepository) FindByUsername(ctx context.Context, username string) (*user.User, error) {
	key := normalizeUsernameKey(username)
	if key == "" {
		return r.repo.FindByUsername(ctx, username)
	}
	if cached, ok := r.usernameCache.Get(key); ok {
		return cached, nil
	}

	u, err := r.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}

	r.usernameCache.Set(key, u)
	r.telegramCache.Set(u.TelegramID, u)

	return u, nil
}

func normalizeUsernameKey(username string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(username), "@"))
}
