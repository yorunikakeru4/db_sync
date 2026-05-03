package service

import (
	"context"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/view"
)

// EmailService handles email-related events and updates the MongoDB read model.
type EmailService struct {
	mongoEmailViewRepo storage.EmailViewRepository
}

// NewEmailService creates a new EmailService.
func NewEmailService(
	mongoEmailViewRepo storage.EmailViewRepository,
) *EmailService {
	return &EmailService{
		mongoEmailViewRepo: mongoEmailViewRepo,
	}
}

// HandleEmailAddedToUser projects an email_added event into the user's important contacts list.
func (es *EmailService) HandleEmailAddedToUser(
	ctx context.Context,
	event events.EmailAddedPayload,
) error {
	email := view.ImportantContactView{
		Email:      event.Address,
		Category:   event.Category,
		Importance: event.Importance,
	}
	return es.mongoEmailViewRepo.AddEmailToUser(ctx, event.UserID, email)
}

// HandleEmailUpdatedForUser projects an email_updated event by updating the contact's
// category and importance in the user's contacts list.
func (es *EmailService) HandleEmailUpdatedForUser(
	ctx context.Context, event events.EmailUpdatedPayload,
) error {
	email := view.ImportantContactView{
		Email:      event.Address,
		Category:   event.NewCategory,
		Importance: event.NewImportance,
	}
	return es.mongoEmailViewRepo.UpdateEmailForUser(ctx, event.UserID, email)
}

// HandleEmailRemovedFromUser projects an email_removed event by removing the contact
// from the user's contacts list.
func (es *EmailService) HandleEmailRemovedFromUser(
	ctx context.Context, event events.EmailRemovedPayload,
) error {
	return es.mongoEmailViewRepo.RemoveEmailFromUser(ctx, event.UserID, event.Address)
}
