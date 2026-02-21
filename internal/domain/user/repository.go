package user

import (
	"context"
)

type Repository interface {
	FindByTelegramID(ctx context.Context, telegramID int64) (*User, error)
}
