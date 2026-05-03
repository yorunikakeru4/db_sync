package events

import "time"

const (
	// UserCreated is emitted after a user insert succeeds.
	UserCreated = "user_created"
	// UserDeleted is emitted after a user delete succeeds.
	UserDeleted = "user_deleted"
	// UserUpdated is emitted after a user update succeeds.
	UserUpdated = "user_updated"
)

// UserCreatedPayload is the payload for a user_created event.
type UserCreatedPayload struct {
	// ID is the primary key of the newly created user.
	ID int64 `json:"id"`
	// Email is the user's email address.
	Email string `json:"email"`
	// CreatedAt is the UTC timestamp when the user was created.
	CreatedAt time.Time `json:"created_at"`
}

// UserDeletedPayload is the payload for a user_deleted event.
type UserDeletedPayload struct {
	// ID is the primary key of the deleted user.
	ID int64 `json:"id"`
}

// UserUpdatedPayload is the payload for a user_updated event.
type UserUpdatedPayload struct {
	// ID is the primary key of the updated user.
	ID int64 `json:"id"`
	// Email is the user's email address after the update.
	Email string `json:"email"`
}
