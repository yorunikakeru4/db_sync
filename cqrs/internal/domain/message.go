package domain

import "time"

// Message represents a message row in the PostgreSQL messages table.
type Message struct {
	// ID is the primary key.
	ID int `db:"id"`
	// ExternalID is an external reference identifier.
	ExternalID string `db:"external_id"`
	// SenderID is the foreign key referencing the sending user.
	SenderID int `db:"sender_id"`
	// ReceiverID is the foreign key referencing the receiving user.
	ReceiverID int `db:"receiver_id"`
	// Subject is the message subject line.
	Subject string `db:"subject"`
	// Text is the message body.
	Text string `db:"text"`
	// DateSent is the UTC timestamp when the message was sent.
	DateSent time.Time `db:"date_sent"`
	// CreatedAt is the UTC timestamp when the record was created.
	CreatedAt time.Time `db:"created_at"`
}

// MessageFile represents a file attachment associated with a message.
type MessageFile struct {
	// ID is the primary key.
	ID int `db:"id"`
	// MessageID is the foreign key referencing the parent message.
	MessageID int `db:"message_id"`
	// FilePath is the storage path of the file.
	FilePath string `db:"file_path"`
	// Type is the MIME type of the file.
	Type string `db:"type"`
	// Size is the file size in bytes.
	Size int `db:"size"`
	// CreatedAt is the UTC timestamp when the attachment was created.
	CreatedAt time.Time `db:"created_at"`
}

// MessageStatus represents a delivery/read status entry for a message.
type MessageStatus struct {
	// ID is the primary key.
	ID int `db:"id"`
	// MessageID is the foreign key referencing the parent message.
	MessageID int `db:"message_id"`
	// Status is the snake_case status value (e.g. sent, delivered, read).
	Status string `db:"status"`
	// UpdatedAt is the UTC timestamp of the last status change.
	UpdatedAt time.Time `db:"updated_at"`
}
