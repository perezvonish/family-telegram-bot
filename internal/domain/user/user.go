package user

type User struct {
	TelegramID int64  `json:"telegramId"`
	FirstName  string `json:"firstName"`
	Username   string `json:"username"`
}

func NewUser(telegramID int64, firstName, username string) *User {
	return &User{
		TelegramID: telegramID,
		FirstName:  firstName,
		Username:   username,
	}
}
