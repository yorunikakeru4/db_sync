// Package models defines the data structures used in the application layer.
package models

import "time"

// User is the enriched application-layer user model that combines the write-model
// user with their denormalised important contacts and messages.
type User struct {
	// ID is the user's primary key.
	ID int
	// Email is the user's email address.
	Email string
	// ImportantContacts is the list of contacts marked as important by the user.
	ImportantContacts []Contact
	// Messages is the list of messages associated with the user.
	Messages []Message
	// CreatedAt is the UTC timestamp when the user was created.
	CreatedAt time.Time
}
