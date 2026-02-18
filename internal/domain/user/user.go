package user

import (
	"github.com/google/uuid"
)

type User struct {
	Id uuid.UUID `json:"id"`

	TelegramId int64 `json:"telegramId"`

	FirstName string `json:"firstName"`
	Username  string `json:"username"`
}
