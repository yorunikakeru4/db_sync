package view

import "time"

type (
	MessageView struct {
		ID        int       `bson:"id"`
		Subject   string    `bson:"subject"`
		Text      string    `bson:"text"`
		CreatedAt time.Time `bson:"created_at"`
	}
)
