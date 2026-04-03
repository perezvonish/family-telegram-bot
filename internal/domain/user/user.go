package user

import "github.com/google/uuid"

type User struct {
	ID         uuid.UUID `json:"id"`
	MongoID    string    `json:"mongoId"`
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

func (u *User) PrimaryStorageID() string {
	if u == nil {
		return ""
	}
	return u.MongoID
}
