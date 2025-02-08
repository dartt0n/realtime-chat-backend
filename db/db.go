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

type CreateUser struct {
	Email   string
	PwdHash string
}
