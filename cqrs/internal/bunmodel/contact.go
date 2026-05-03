package bunmodel

import (
	"time"

	"github.com/uptrace/bun"
)

// Contact represents a contact record in the PostgreSQL contacts table.
type Contact struct {
	bun.BaseModel `bun:"table:contacts,alias:c"`

	// ID is the primary key.
	ID int64 `bun:"id,pk,autoincrement" yaml:"id"`
	// Value is the contact value string.
	Value string `bun:"contact_value,notnull" yaml:"contact_value"`
	// CreatedAt is the UTC timestamp when the record was created.
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" yaml:"created_at"`
}

// UserContact represents the association between a user and a contact.
type UserContact struct {
	bun.BaseModel `bun:"table:users_contacts,alias:uc"`

	// ID is the primary key.
	ID int64 `bun:"id,pk,autoincrement" yaml:"id"`
	// UserID is the foreign key referencing the owner user.
	UserID int64 `bun:"user_id,notnull" yaml:"user_id"`
	// ContactID is the foreign key referencing the contact record.
	ContactID int64 `bun:"contact_id,notnull" yaml:"contact_id"`
	// Importance is the user-assigned importance level for this contact.
	Importance int `bun:"importance,notnull,default:0" yaml:"importance"`
	// Category is the user-assigned category identifier for this contact.
	Category int `bun:"category,notnull,default:0" yaml:"category"`
	// CreatedAt is the UTC timestamp when the association was created.
	CreatedAt time.Time `bun:"created_at,notnull,default:now()" yaml:"created_at"`
}
