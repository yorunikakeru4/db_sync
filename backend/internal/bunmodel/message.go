package bunmodel

import (
	"time"

	"github.com/uptrace/bun"
)

// Message represents a message row in the PostgreSQL messages table.
type Message struct {
	bun.BaseModel `bun:"table:messages,alias:m"`

	// ID is the primary key.
	ID int64 `bun:"id,pk,autoincrement"`
	// ExternalID is an external reference identifier.
	ExternalID string `bun:"external_id,notnull"`
	// SenderID is the foreign key referencing the sending user.
	SenderID int64 `bun:"sender_id,notnull"`
	// ReceiverID is the foreign key referencing the receiving user.
	ReceiverID int64 `bun:"receiver_id,notnull"`
	// Subject is the message subject line.
	Subject string `bun:"subject"`
	// Text is the message body.
	Text string `bun:"text"`
	// DateSent is the UTC timestamp when the message was sent.
	DateSent time.Time `bun:"date_sent"`
	// CreatedAt is the UTC timestamp when the record was created.
	CreatedAt time.Time `bun:"created_at,notnull,default:now()"`
}
