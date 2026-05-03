package dbfixture

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
)

// UserNotificationLoader loads user notification fixtures.
type UserNotificationLoader struct {
	pgLoaderBase[seedinput.UserNotification, models.UserNotification]
}

// NewUserNotificationLoader creates a Loader for user notifications.
func NewUserNotificationLoader(f *Fixture, params LoaderParams) *UserNotificationLoader {
	return &UserNotificationLoader{pgLoaderBase: pgLoaderBase[seedinput.UserNotification, models.UserNotification]{
		params: params, fixture: f, name: ModelUserNotification,
	}}
}

func (l *UserNotificationLoader) Resolve(ctx context.Context, input *seedinput.UserNotification) error {
	user, ok := Get[*models.User](l.fixture, input.UserKey)
	if !ok {
		return fmt.Errorf("user %q not found", input.UserKey)
	}
	input.UserID = user.ID
	return nil
}

func (l *UserNotificationLoader) Defaults(ctx context.Context, input *seedinput.UserNotification, fake bool) error {
	if input.CreatedAt.IsZero() {
		input.CreatedAt = unixtime.Nano(time.Now().Add(-2 * time.Hour).UnixNano())
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	if input.ExpiresAt.IsZero() {
		input.ExpiresAt = unixtime.Nano(time.Now().Add(90 * 24 * time.Hour).UnixNano())
	}
	return nil
}

func (l *UserNotificationLoader) PopulateModel(ctx context.Context, model *models.UserNotification, input *seedinput.UserNotification) ([]string, error) {
	var cols []string
	if input.UserID != 0 && model.UserID != input.UserID {
		model.UserID = input.UserID
		cols = append(cols, "user_id")
	}
	model.Type = input.Type
	model.Severity = input.Severity
	model.Title = input.Title
	model.Message = input.Message
	model.DedupeKey = input.DedupeKey
	cols = append(cols, "type", "severity", "title", "message", "dedupe_key")
	if input.ActionTitle != nil {
		model.ActionTitle = *input.ActionTitle
		cols = append(cols, "action_title")
	}
	if input.ActionURL != nil {
		model.ActionURL = *input.ActionURL
		cols = append(cols, "action_url")
	}
	if !input.ReadAt.IsZero() {
		model.ReadAt = input.ReadAt
		cols = append(cols, "read_at")
	}
	if !input.ExpiresAt.IsZero() {
		model.ExpiresAt = input.ExpiresAt
		cols = append(cols, "expires_at")
	}
	if !input.CreatedAt.IsZero() {
		model.CreatedAt = input.CreatedAt
		cols = append(cols, "created_at")
	}
	if !input.UpdatedAt.IsZero() {
		model.UpdatedAt = input.UpdatedAt
		cols = append(cols, "updated_at")
	}
	return cols, nil
}

func (l *UserNotificationLoader) Validate(ctx context.Context, model *models.UserNotification) error {
	return model.Validate()
}
