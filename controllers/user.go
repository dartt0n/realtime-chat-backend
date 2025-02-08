package controllers

import (
	"github.com/dartt0n/realtime-chat-backend/forms"
	"github.com/dartt0n/realtime-chat-backend/service"

	"net/http"

	"github.com/gin-gonic/gin"
)

// UserController ...
type UserController struct{}

func NewUserController() *UserController {
	return &UserController{}
}

var userForm = new(forms.UserForm)

// getUserID ...
func getUserID(c *gin.Context) (userID int64) {
	//MustGet returns the value for the given key if it exists, otherwise it panics.
	return c.MustGet("userID").(int64)
}

// Login ...
func (ctrl UserController) Login(c *gin.Context) {
	var loginForm forms.LoginForm

	if validationErr := c.ShouldBindJSON(&loginForm); validationErr != nil {
		message := userForm.Login(validationErr)
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": message})
		return
	}

	_, token, err := service.User.Login(loginForm)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": "Invalid login details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// Register ...
func (ctrl UserController) Register(c *gin.Context) {
	var registerForm forms.RegisterForm

	if err := c.ShouldBindJSON(&registerForm); err != nil {
		message := userForm.Register(err)
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": message})
		return
	}

	_, err := service.User.Register(registerForm)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, gin.H{"message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

// Logout ...
func (ctrl UserController) Logout(c *gin.Context) {

	au, err := service.Auth.ExtractTokenMetadata(c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"message": "User not logged in"})
		return
	}

	_, delErr := service.Auth.DeleteAuth(au.AccessUUID)
	if delErr != nil { // if any goes wrong
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}
