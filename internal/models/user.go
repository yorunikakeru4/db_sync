package models

import "time"

type User struct {
	ID              int
	Email           string
	ImportantEmails []Email
	Messages        []Message
	CreatedAt       time.Time
}
