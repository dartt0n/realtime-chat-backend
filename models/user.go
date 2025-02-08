package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID        UserID `json:"id" bson:"_id"`
	CreatedAt int64  `json:"-" bson:"created_at"`
	UpdatedAt int64  `json:"-" bson:"updated_at"`
	DeletedAt int64  `json:"-" bson:"deleted_at"`

	Email    string `json:"email" bson:"email"`
	Password string `json:"-"`
}

type UserID bson.ObjectID

func ParseUserID(id string) (UserID, error) {
	uid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return UserID{}, err
	}

	return UserID(uid), nil
}

func (id UserID) String() string {
	return bson.ObjectID(id).Hex()
}
