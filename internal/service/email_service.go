package service

import (
	"context"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/view"
)

type EmailService struct {
	postgresEmailRepo  storage.EmailRepository
	mongoEmailViewRepo storage.EmailViewRepository
}

func NewEmailService(
	postgresEmailRepo storage.EmailRepository,
	mongoEmailViewRepo storage.EmailViewRepository,
) *EmailService {
	return &EmailService{
		postgresEmailRepo:  postgresEmailRepo,
		mongoEmailViewRepo: mongoEmailViewRepo,
	}
}

func (es *EmailService) HandleEmailAddedToUser(
	ctx context.Context,
	event events.EmailAddedPayload,
) error {
	email := view.ImportantContactView{
		Email:      event.Address,
		Category:   event.Category,
		Importance: event.Importance,
	}
	userID := event.UserID
	return es.mongoEmailViewRepo.AddEmailToUser(ctx, userID, email)
}

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

func (es *EmailService) HandleEmailRemovedFromUser(
	ctx context.Context, event events.EmailRemovedPayload,
) error {
	return es.mongoEmailViewRepo.RemoveEmailFromUser(ctx, event.UserID, event.Address)
}
