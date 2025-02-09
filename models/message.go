package models

import "time"

type Message struct {
	Author    string    `json:"author" bson:"from"`
	Text      string    `json:"text" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"createdat"`
}
