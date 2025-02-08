package db

import (
	"github.com/dartt0n/realtime-chat-backend/models"
)

type Database interface {
	EmailExists(email string) (bool, error)
	FindByEmail(email string) (models.User, error)

	GetUser(id models.UserID) (models.User, error)
	CreateUser(user CreateUser) (models.User, error)
}

var _globalDB Database

func GetDB() Database {
	return _globalDB
}

type CreateUser struct {
	Email   string
	PwdHash string
}
