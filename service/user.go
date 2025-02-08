package service

import (
	"errors"
	"log"

	"github.com/dartt0n/realtime-chat-backend/db"
	"github.com/dartt0n/realtime-chat-backend/forms"
	"github.com/dartt0n/realtime-chat-backend/models"
	"golang.org/x/crypto/bcrypt"
)

type userService struct{}

var User = new(userService)

func (s userService) Login(form forms.LoginForm) (user models.User, token models.Token, err error) {
	user, err = db.GetDB().FindByEmail(form.Email)
	if err != nil {
		return user, token, err
	}

	bytePassword := []byte(form.Password)
	byteHashedPassword := []byte(user.Password)

	err = bcrypt.CompareHashAndPassword(byteHashedPassword, bytePassword)

	if err != nil {
		return user, token, err
	}

	tokenDetails, err := Auth.CreateToken(user.ID)
	if err != nil {
		return user, token, err
	}

	saveErr := Auth.CreateAuth(user.ID, tokenDetails)
	if saveErr == nil {
		token.AccessToken = tokenDetails.AccessToken
		token.RefreshToken = tokenDetails.RefreshToken
	}

	return user, token, nil
}

// Register ...
func (s userService) Register(form forms.RegisterForm) (user models.User, err error) {
	exists, err := db.GetDB().EmailExists(form.Email)
	if err != nil {
		log.Printf("failed to check if email exists: %v", err)
		return user, errors.New("something went wrong, please try again later")
	}
	if exists {
		return user, errors.New("email already exists")
	}

	bytePassword := []byte(form.Password)
	hashedPassword, err := bcrypt.GenerateFromPassword(bytePassword, bcrypt.DefaultCost)
	if err != nil {
		log.Printf("failed to hash password: %v", err)
		return user, errors.New("something went wrong, please try again later")
	}

	//Create the user and return back the user ID
	user, err = db.GetDB().CreateUser(db.CreateUser{
		Email:   form.Email,
		PwdHash: string(hashedPassword),
	})

	if err != nil {
		return user, errors.New("something went wrong, please try again later")
	}

	return user, err
}

// One ...
func (s userService) One(userID models.UserID) (user models.User, err error) {
	user, err = db.GetDB().GetUser(userID)
	return user, err
}
