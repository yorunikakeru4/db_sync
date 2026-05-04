package service

import (
	"context"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/view"
)

// ContactService handles contact-related events and updates the MongoDB read model.
type ContactService struct {
	mongoContactViewRepo storage.ContactViewRepository
}

// NewContactService creates a new ContactService.
func NewContactService(
	mongoContactViewRepo storage.ContactViewRepository,
) *ContactService {
	return &ContactService{
		mongoContactViewRepo: mongoContactViewRepo,
	}
}

// HandleContactAddedToUser projects a contact_added event into the user's important contacts list.
func (es *ContactService) HandleContactAddedToUser(
	ctx context.Context,
	event events.ContactAddedPayload,
) error {
	contact := view.ImportantContactView{
		ContactID:  event.ContactID,
		Value:      event.Value,
		Category:   event.Category,
		Importance: event.Importance,
	}
	return es.mongoContactViewRepo.AddContactToUser(ctx, event.UserID, contact)
}

// HandleContactUpdatedForUser projects a contact_updated event by updating the contact's
// value, category and importance in the user's contacts list.
func (es *ContactService) HandleContactUpdatedForUser(
	ctx context.Context, event events.ContactUpdatedPayload,
) error {
	contact := view.ImportantContactView{
		ContactID:  event.ContactID,
		Value:      event.Value,
		Category:   event.NewCategory,
		Importance: event.NewImportance,
	}
	return es.mongoContactViewRepo.UpdateContactForUser(ctx, event.UserID, event.ContactID, contact)
}

// HandleContactRemovedFromUser projects a contact_removed event by removing the contact
// from the user's contacts list.
func (es *ContactService) HandleContactRemovedFromUser(
	ctx context.Context, event events.ContactRemovedPayload,
) error {
	return es.mongoContactViewRepo.RemoveContactFromUser(ctx, event.UserID, event.ContactID)
}
