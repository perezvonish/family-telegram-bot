package user

import "github.com/google/uuid"

type User struct {
	ID         uuid.UUID `json:"id"`
	TelegramID int64     `json:"telegramId"`
	FirstName  string    `json:"firstName"`
	Username   string    `json:"username"`
}

func NewUser(telegramID int64, firstName, username string) *User {
	return &User{
		TelegramID: telegramID,
		FirstName:  firstName,
		Username:   username,
	}
}
