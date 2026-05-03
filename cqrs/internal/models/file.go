package models

import "time"

// File represents a file attachment associated with a message.
type File struct {
	// ID is the primary key.
	ID int
	// MessageID is the foreign key referencing the parent message.
	MessageID int
	// FileID is the external storage identifier for the file.
	FileID string
	// Type is the MIME type of the file.
	Type string
	// Size is the file size in bytes.
	Size int64
	// CreatedAt is the UTC timestamp when the attachment was created.
	CreatedAt time.Time
}
