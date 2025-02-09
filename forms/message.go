package forms

import (
	"encoding/json"

	"github.com/go-playground/validator/v10"
)

// MessageForm represents the base form structure for message-related forms
type MessageForm struct{}

// TextMessage represents a text message with content
// Content must be between 1 and 4096 characters
type TextMessage struct {
	Content string `form:"content" json:"content" binding:"required,min=1,max=4096"`
}

// Content returns the appropriate error message for content validation tags
func (f MessageForm) Content(tag string, errMsg ...string) string {
	switch tag {
	case "required":
		return "Please provide message content"
	case "min", "max":
		return "Message content can be from 1 to 4096 characters"
	default:
		return "Something went wrong, please try again later"
	}
}

// Text validates a TextMessagew and returns appropriate error messages
func (f MessageForm) Text(err error) string {
	switch err.(type) {
	case validator.ValidationErrors:
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			return "Something went wrong, please try again later"
		}

		for _, err := range err.(validator.ValidationErrors) {
			if err.Field() == "Content" {
				return f.Content(err.Tag())
			}
		}
	default:
		return "Invalid request"
	}
	return "Something went wrong, please try again later"
}
