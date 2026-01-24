package events

import "time"

type UserCreatedPayload struct {
	ID        int       `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type UserDeletedPayload struct {
	ID int `json:"id"`
}
