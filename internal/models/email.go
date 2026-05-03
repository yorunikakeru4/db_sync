// Package models defines the data structures used in the application.
package models

// Email represents an important contact associated with a user.
type Email struct {
	// ID is the primary key of the email record.
	ID int
	// Address is the email address string.
	Address string
	// Category is the user-assigned category label for this contact.
	Category string
	// Importance is the user-assigned importance level for this contact.
	Importance int
	// CreatedAt is the formatted timestamp when the association was created.
	CreatedAt string
}
