package repository

import (
	"context"
	"errors"
	"perezvonish/health-tracker/internal/domain/user"
	"perezvonish/health-tracker/internal/infrastructure/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

const usersCollection = "users"

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	collection *mongo.Collection
}

func NewUserRepository(db *database.MongoDB) *UserRepository {
	return &UserRepository{
		collection: db.Collection(usersCollection),
	}
}

func (r *UserRepository) FindByTelegramID(ctx context.Context, telegramID int64) (*user.User, error) {
	var model UserModel

	err := r.collection.FindOne(ctx, bson.M{"telegram_id": telegramID}).Decode(&model)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return model.ToEntity(), nil
}
