package repository

import (
	"perezvonish/health-tracker/internal/domain/user"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserModel struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	TelegramID int64              `bson:"telegram_id"`
	FirstName  string             `bson:"first_name"`
	Username   string             `bson:"username"`
}

func (m *UserModel) ToEntity() *user.User {
	return &user.User{
		TelegramID: m.TelegramID,
		FirstName:  m.FirstName,
		Username:   m.Username,
	}
}

func ToUserModel(entity *user.User) *UserModel {
	return &UserModel{
		TelegramID: entity.TelegramID,
		FirstName:  entity.FirstName,
		Username:   entity.Username,
	}
}
