package models

import "time"

// Message is the application-layer message model enriched with statuses and file attachments.
type Message struct {
	// ID is the message's primary key.
	ID int
	// ExternalID is the external reference identifier.
	ExternalID string
	// SenderID is the primary key of the sending user.
	SenderID int
	// ReceiverID is the primary key of the receiving user.
	ReceiverID int
	// Subject is the message subject line.
	Subject string
	// Text is the message body.
	Text string
	// DateSent is the formatted date when the message was sent.
	DateSent string
	// CreatedAt is the UTC timestamp when the record was created.
	CreatedAt time.Time
	// Statuses is the ordered list of delivery/read status entries.
	Statuses []Status
	// Files is the list of file attachments for this message.
	Files []File
}

// Status represents a delivery or read status entry for a message.
type Status struct {
	// ID is the primary key.
	ID int
	// MessageID is the foreign key referencing the parent message.
	MessageID int
	// Status is the snake_case status value (e.g. sent, delivered, read).
	Status string
	// UpdatedAt is the UTC timestamp of the last status change.
	UpdatedAt time.Time
}
