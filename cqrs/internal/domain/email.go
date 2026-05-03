package domain

import "time"

// Email represents an email address record in the PostgreSQL emails table.
type Email struct {
	// ID is the primary key.
	ID int `db:"id"`
	// Address is the email address string.
	Address string `db:"email_address"`
	// CreatedAt is the UTC timestamp when the record was created.
	CreatedAt time.Time `db:"created_at"`
}

// UserEmail represents the association between a user and an email address
// in the PostgreSQL users_emails join table.
type UserEmail struct {
	// ID is the primary key.
	ID int `db:"id"`
	// UserID is the foreign key referencing the owner user.
	UserID int `db:"user_id"`
	// EmailID is the foreign key referencing the email address.
	EmailID int `db:"email_id"`
	// Importance is the user-assigned importance level for this contact.
	Importance int `db:"importance"`
	// Category is the user-assigned category identifier for this contact.
	Category int `db:"category"`
	// CreatedAt is the UTC timestamp when the association was created.
	CreatedAt time.Time `db:"created_at"`
}
