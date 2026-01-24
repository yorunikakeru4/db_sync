package events

import "time"

type EmailAddedPayload struct {
	EmailID    int       `json:"email_id"`
	UserID     int       `json:"user_id"`
	Address    string    `json:"address"`
	Category   string    `json:"category"`
	Importance int       `json:"importance"`
	CreatedAt  time.Time `json:"created_at"`
}

type EmailUpdatedPayload struct {
	EmailID       int       `json:"email_id"`
	Address       string    `json:"address"`
	UserID        int       `json:"user_id"`
	OldCategory   string    `json:"old_category"`
	NewCategory   string    `json:"new_category"`
	OldImportance int       `json:"old_importance"`
	NewImportance int       `json:"new_importance"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type EmailRemovedPayload struct {
	EmailID int    `json:"email_id"`
	Address string `json:"address"`
	UserID  int    `json:"user_id"`
}
