package events

import "time"

type MessageCreatedPayload struct {
	MessageID int       `json:"message_id"`
	UserID    int       `json:"user_id"`
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	DateSent  time.Time `json:"date_sent"`
}

type MessageDeletedPayload struct {
	MessageID int `json:"message_id"`
	UserID    int `json:"user_id"`
}
