package events

import "time"

const (
	// MessageCreated is emitted after a message insert succeeds.
	MessageCreated = "message_created"
	// MessageDeleted is emitted after a message delete succeeds.
	MessageDeleted = "message_deleted"
)

// MessageCreatedPayload is the payload for a message_created event.
type MessageCreatedPayload struct {
	// MessageID is the primary key of the newly created message.
	MessageID int64 `json:"message_id"`
	// UserID is the primary key of the user who sent the message.
	UserID int64 `json:"user_id"`
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
	MessageID int64 `json:"message_id"`
	// UserID is the primary key of the user who owned the message.
	UserID int64 `json:"user_id"`
}
