package forms

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
)

// UserForm represents the base form structure for user-related forms
type UserForm struct{}

// LoginForm contains the fields required for user login
type LoginForm struct {
	Email    string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required,min=3,max=50"`
}

// RegisterForm contains the fields required for user registration
type RegisterForm struct {
	Email    string `form:"email" json:"email" binding:"required,email"`
	Password string `form:"password" json:"password" binding:"required,min=3,max=50"`
}

// Email validates and returns appropriate error messages for email field validation
func (f UserForm) Email(tag string, errMsg ...string) (message string) {
	switch tag {
	case "required":
		if len(errMsg) == 0 {
			return "Please enter your email"
		}
		return errMsg[0]
	case "min", "max", "email":
		return "Please enter a valid email"
	default:
		return "Something went wrong, please try again later"
	}
}

// Password validates and returns appropriate error messages for password field validation
func (f UserForm) Password(tag string) (message string) {
	switch tag {
	case "required":
		return "Please enter your password"
	case "min", "max":
		return "Your password should be between 3 and 50 characters"
	case "eqfield":
		return "Your passwords does not match"
	default:
		return "Something went wrong, please try again later"
	}
}

// Login validates the login form and returns appropriate error messages
func (f UserForm) Login(err error) string {
	switch err.(type) {
	case validator.ValidationErrors:

		if _, ok := err.(*json.UnmarshalTypeError); ok {
			return "Something went wrong, please try again later"
		}

		for _, err := range err.(validator.ValidationErrors) {
			if err.Field() == "Email" {
				return f.Email(err.Tag())
			}
			if err.Field() == "Password" {
				return f.Password(err.Tag())
			}
		}

	default:
		return "Invalid request"
	}

	return "Something went wrong, please try again later"
}

// Register validates the registration form and returns appropriate error messages
func (f UserForm) Register(err error) string {
	switch err.(type) {
	case validator.ValidationErrors:

		if _, ok := err.(*json.UnmarshalTypeError); ok {
			return "Something went wrong, please try again later"
		}

		for _, err := range err.(validator.ValidationErrors) {
			if err.Field() == "Email" {
				return f.Email(err.Tag())
			}

			if err.Field() == "Password" {
				return f.Password(err.Tag())
			}

		}
	default:
		return "Invalid request"
	}

	return "Something went wrong, please try again later"
}
