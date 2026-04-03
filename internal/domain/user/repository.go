package user

import (
	"context"
)

type Repository interface {
	FindByTelegramID(ctx context.Context, telegramID int64) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	FindAll(ctx context.Context) ([]*User, error)
}
