package events

import "time"

const (
	// ContactAdded is emitted after a contact association insert succeeds.
	ContactAdded = "contact_added"
	// ContactUpdated is emitted after a contact association update succeeds.
	ContactUpdated = "contact_updated"
	// ContactRemoved is emitted after a contact association delete succeeds.
	ContactRemoved = "contact_removed"
)

// ContactAddedPayload is the payload for a contact_added event.
type ContactAddedPayload struct {
	// ContactID is the primary key of the contact record.
	ContactID int64 `json:"contact_id"`
	// UserID is the primary key of the user this contact is associated with.
	UserID int64 `json:"user_id"`
	// Value is the contact value string.
	Value string `json:"value"`
	// Category is the user-assigned category label for this contact.
	Category string `json:"category"`
	// Importance is the user-assigned importance level.
	Importance int `json:"importance"`
	// CreatedAt is the UTC timestamp when the association was created.
	CreatedAt time.Time `json:"created_at"`
}

// ContactUpdatedPayload is the payload for a contact_updated event.
type ContactUpdatedPayload struct {
	// ContactID is the primary key of the contact record.
	ContactID int64 `json:"contact_id"`
	// Value is the contact value string.
	Value string `json:"value"`
	// OldValue is the contact value string before the update.
	OldValue string `json:"old_value"`
	// UserID is the primary key of the owning user.
	UserID int64 `json:"user_id"`
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

// ContactRemovedPayload is the payload for a contact_removed event.
type ContactRemovedPayload struct {
	// ContactID is the primary key of the removed contact record.
	ContactID int64 `json:"contact_id"`
	// Value is the contact value string that was removed.
	Value string `json:"value"`
	// UserID is the primary key of the owning user.
	UserID int64 `json:"user_id"`
}
