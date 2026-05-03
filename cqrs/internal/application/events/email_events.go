package events

import "time"

// EmailAddedPayload is the payload for an email_added event.
type EmailAddedPayload struct {
	// EmailID is the primary key of the email record.
	EmailID int `json:"email_id"`
	// UserID is the primary key of the user this email is associated with.
	UserID int `json:"user_id"`
	// Address is the email address string.
	Address string `json:"address"`
	// Category is the user-assigned category label for this contact.
	Category string `json:"category"`
	// Importance is the user-assigned importance level.
	Importance int `json:"importance"`
	// CreatedAt is the UTC timestamp when the association was created.
	CreatedAt time.Time `json:"created_at"`
}

// EmailUpdatedPayload is the payload for an email_updated event.
type EmailUpdatedPayload struct {
	// EmailID is the primary key of the email record.
	EmailID int `json:"email_id"`
	// Address is the email address string.
	Address string `json:"address"`
	// UserID is the primary key of the owning user.
	UserID int `json:"user_id"`
	// OldCategory is the category value before the update.
	OldCategory string `json:"old_category"`
	// NewCategory is the category value after the update.
	NewCategory string `json:"new_category"`
	// OldImportance is the importance value before the update.
	OldImportance int `json:"old_importance"`
	// NewImportance is the importance value after the update.
	NewImportance int `json:"new_importance"`
	// UpdatedAt is the UTC timestamp of the update.
	UpdatedAt time.Time `json:"updated_at"`
}

// EmailRemovedPayload is the payload for an email_removed event.
type EmailRemovedPayload struct {
	// EmailID is the primary key of the removed email record.
	EmailID int `json:"email_id"`
	// Address is the email address string that was removed.
	Address string `json:"address"`
	// UserID is the primary key of the owning user.
	UserID int `json:"user_id"`
}
