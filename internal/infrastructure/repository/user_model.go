package repository

import (
	"perezvonish/health-tracker/internal/domain/user"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type FlexInt64 int64

func (f *FlexInt64) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	rv := bson.RawValue{Type: t, Value: data}

	switch t {
	case bsontype.Int32:
		*f = FlexInt64(rv.Int32())
	case bsontype.Int64:
		*f = FlexInt64(rv.Int64())
	case bsontype.Double:
		*f = FlexInt64(rv.Double())
	default:
		*f = 0
	}
	return nil
}

var mongoNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

func objectIDToUUID(oid primitive.ObjectID) uuid.UUID {
	return uuid.NewSHA1(mongoNamespace, oid[:])
}

type UserModel struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	TelegramID FlexInt64          `bson:"telegram_id"`
	FirstName  string             `bson:"first_name"`
	Username   string             `bson:"username"`
}

func (m *UserModel) ToEntity() *user.User {
	return &user.User{
		ID:         objectIDToUUID(m.ID),
		MongoID:    m.ID.Hex(),
		TelegramID: int64(m.TelegramID),
		FirstName:  m.FirstName,
		Username:   m.Username,
	}
}

func ToUserModel(entity *user.User) *UserModel {
	return &UserModel{
		TelegramID: FlexInt64(entity.TelegramID),
		FirstName:  entity.FirstName,
		Username:   entity.Username,
	}
}
