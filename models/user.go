package models

import (
	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID       UserID `json:"id" bson:"_id"`
	Email    string `json:"email" bson:"email"`
	Username string `json:"username" bson:"username"`
	Password string `json:"-"`
}

type UserID string

func ParseUserID(id string) (UserID, error) {
	uid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return "", err
	}

	return UserID(uid.Hex()), nil
}

func (id UserID) String() string {
	return string(id)
}
