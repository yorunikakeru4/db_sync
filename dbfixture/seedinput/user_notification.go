package seedinput

import (
	"github.com/uptrace/bun"

	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// UserNotification represents a user notification fixture input.
type UserNotification struct {
	bun.BaseModel `bun:"user_notifications,alias:un"`

	Key string `yaml:"key" json:"key" bun:"-"`

	UserKey string `yaml:"user_key" json:"userKey" bun:"-"`
	UserID  uint64 `yaml:"-" json:"-" bun:",nullzero"`

	Type      models.UserNotificationType     `yaml:"type" json:"type"`
	Severity  models.UserNotificationSeverity `yaml:"severity" json:"severity"`
	Title     string                          `yaml:"title" json:"title"`
	Message   string                          `yaml:"message" json:"message"`
	ActionTitle *string                       `yaml:"action_title" json:"actionTitle"`
	ActionURL   *string                       `yaml:"action_url" json:"actionUrl"`
	DedupeKey string                          `yaml:"dedupe_key" json:"dedupeKey"`

	ReadAt    unixtime.Nano `yaml:"read_at" json:"readAt"`
	ExpiresAt unixtime.Nano `yaml:"expires_at" json:"expiresAt"`
	CreatedAt unixtime.Nano `yaml:"created_at" json:"createdAt"`
	UpdatedAt unixtime.Nano `yaml:"updated_at" json:"updatedAt"`
}

// FixtureKey returns the fixture key for this user notification.
func (n *UserNotification) FixtureKey() string { return n.Key }

// Validate validates the user notification input.
func (n *UserNotification) Validate() error {
	if n.Key == "" {
		return validerr.Empty("key")
	}
	if n.UserKey == "" {
		return validerr.Empty("user_key")
	}
	if n.Type == "" {
		return validerr.Empty("type")
	}
	if n.Severity == "" {
		return validerr.Empty("severity")
	}
	if n.Title == "" {
		return validerr.Empty("title")
	}
	if n.Message == "" {
		return validerr.Empty("message")
	}
	if n.DedupeKey == "" {
		return validerr.Empty("dedupe_key")
	}
	return nil
}
