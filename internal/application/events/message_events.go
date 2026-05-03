package events

import "time"

// MessageCreatedPayload is the payload for a message_created event.
type MessageCreatedPayload struct {
	// MessageID is the primary key of the newly created message.
	MessageID int `json:"message_id"`
	// UserID is the primary key of the user who sent the message.
	UserID int `json:"user_id"`
	// Subject is the message subject line.
	Subject string `json:"subject"`
	// Content is the message body text.
	Content string `json:"content"`
	// DateSent is the UTC timestamp when the message was sent.
	DateSent time.Time `json:"date_sent"`
}

// MessageDeletedPayload is the payload for a message_deleted event.
type MessageDeletedPayload struct {
	// MessageID is the primary key of the deleted message.
	MessageID int `json:"message_id"`
	// UserID is the primary key of the user who owned the message.
	UserID int `json:"user_id"`
}
