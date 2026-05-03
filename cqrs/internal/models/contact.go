// Package models defines the data structures used in the application.
package models

// Contact represents an important contact associated with a user.
type Contact struct {
	// ID is the primary key of the contact record.
	ID int
	// Value is the contact value string.
	Value string
	// Category is the user-assigned category label for this contact.
	Category string
	// Importance is the user-assigned importance level for this contact.
	Importance int
	// CreatedAt is the formatted timestamp when the association was created.
	CreatedAt string
}
