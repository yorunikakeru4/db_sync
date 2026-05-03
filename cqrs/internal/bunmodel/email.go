package bunmodel

import (
	"time"

	"github.com/uptrace/bun"
)

// Email represents an email address record in the PostgreSQL emails table.
type Email struct {
	bun.BaseModel `bun:"table:emails,alias:e"`

	// ID is the primary key.
	ID int64 `bun:"id,pk,autoincrement" yaml:"id"`
	// Address is the email address string.
	Address string `bun:"email_address,notnull" yaml:"email_address"`
	// CreatedAt is the UTC timestamp when the record was created.
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" yaml:"created_at"`
}

// UserEmail represents the association between a user and an email address.
type UserEmail struct {
	bun.BaseModel `bun:"table:users_emails,alias:ue"`

	// ID is the primary key.
	ID int64 `bun:"id,pk,autoincrement" yaml:"id"`
	// UserID is the foreign key referencing the owner user.
	UserID int64 `bun:"user_id,notnull" yaml:"user_id"`
	// EmailID is the foreign key referencing the email address.
	EmailID int64 `bun:"email_id,notnull" yaml:"email_id"`
	// Importance is the user-assigned importance level for this contact.
	Importance int `bun:"importance,notnull,default:0" yaml:"importance"`
	// Category is the user-assigned category identifier for this contact.
	Category int `bun:"category,notnull,default:0" yaml:"category"`
	// CreatedAt is the UTC timestamp when the association was created.
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" yaml:"created_at"`
}
