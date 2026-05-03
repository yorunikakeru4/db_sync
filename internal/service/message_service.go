package service

import (
	"context"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/view"
)

// MessageService handles message-related events and updates the MongoDB read model.
type MessageService struct {
	messageViewRepo storage.MessageViewRepository
}

// NewMessageService creates a new MessageService.
func NewMessageService(
	messageViewRepo storage.MessageViewRepository,
) *MessageService {
	return &MessageService{
		messageViewRepo: messageViewRepo,
	}
}

// HandleMessageAddedToUser projects a message_created event into the user's message list.
func (ms *MessageService) HandleMessageAddedToUser(
	ctx context.Context,
	event events.MessageCreatedPayload,
) error {
	message := view.MessageView{
		ID:        event.MessageID,
		Subject:   event.Subject,
		Text:      event.Content,
		CreatedAt: event.DateSent,
	}
	return ms.messageViewRepo.AddMessageToUser(ctx, event.UserID, message)
}

// HandleMessageDeletedFromUser projects a message_deleted event by removing the message
// from the user's message list.
func (ms *MessageService) HandleMessageDeletedFromUser(
	ctx context.Context,
	event events.MessageDeletedPayload,
) error {
	return ms.messageViewRepo.RemoveMessageFromUser(ctx, event.UserID, event.MessageID)
}
