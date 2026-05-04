// Package events defines the domain event envelope and payload types consumed from Kafka.
package events

import "time"

// UserCreatedPayload is the payload for a user_created event.
type UserCreatedPayload struct {
	// ID is the primary key of the newly created user.
	ID int `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// CreatedAt is the UTC timestamp when the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// UserDeletedPayload is the payload for a user_deleted event.
type UserDeletedPayload struct {
	// ID is the primary key of the deleted user.
	ID int `json:"id"`
}

// UserUpdatedPayload is the payload for a user_updated event.
type UserUpdatedPayload struct {
	// ID is the primary key of the updated user.
	ID int `json:"id"`
	// Email is the user's updated email address.
	Email string `json:"email"`
}
