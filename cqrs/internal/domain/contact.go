package domain

import "time"

// Contact represents a contact record in the PostgreSQL contacts table.
type Contact struct {
	// ID is the primary key.
	ID int `db:"id"`
	// Value is the contact value string.
	Value string `db:"contact_value"`
	// CreatedAt is the UTC timestamp when the record was created.
	CreatedAt time.Time `db:"created_at"`
}

// UserContact represents the association between a user and a contact
// in the PostgreSQL users_contacts join table.
type UserContact struct {
	// ID is the primary key.
	ID int `db:"id"`
	// UserID is the foreign key referencing the owner user.
	UserID int `db:"user_id"`
	// ContactID is the foreign key referencing the contact record.
	ContactID int `db:"contact_id"`
	// Importance is the user-assigned importance level for this contact.
	Importance int `db:"importance"`
	// Category is the user-assigned category identifier for this contact.
	Category int `db:"category"`
	// CreatedAt is the UTC timestamp when the association was created.
	CreatedAt time.Time `db:"created_at"`
}
