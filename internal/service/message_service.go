package service

import (
	"context"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/view"
)

type MessageService struct {
	messageRepo     storage.MessageRepository
	messageViewRepo storage.MessageViewRepository
}

func NewMessageService(
	messageRepo storage.MessageRepository,
	messageViewRepo storage.MessageViewRepository,
) *MessageService {
	return &MessageService{
		messageRepo:     messageRepo,
		messageViewRepo: messageViewRepo,
	}
}

func (ms *MessageService) HandleMessageAddedToUser(
	ctx context.Context,
	event events.MessageCreatedPayload,
) error {
	message := view.MessageView{
		ID:      event.MessageID,
		Subject: event.Subject,
		Text:    event.Content,
	}
	return ms.messageViewRepo.AddMessageToUser(ctx, event.UserID, message)
}

func (ms *MessageService) HandleMessageDeletedFromUser(
	ctx context.Context,
	event events.MessageDeletedPayload,
) error {
	return ms.messageViewRepo.RemoveMessageFromUser(ctx, event.UserID, event.MessageID)
}
