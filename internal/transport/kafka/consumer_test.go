package kafka

import (
	"context"
	"testing"

	"db_sync/internal/application/events"
	"db_sync/internal/service"
	"db_sync/internal/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// syncServiceWith builds a SyncService backed by the provided mock repositories.
func syncServiceWith(
	userViewRepo *mocks.MockUserViewRepository,
	emailViewRepo *mocks.MockEmailViewRepository,
	msgViewRepo *mocks.MockMessageViewRepository,
) *service.SyncService {
	userRepo := new(mocks.MockUserRepository)
	userSvc := service.NewUserService(userRepo, userViewRepo)
	emailSvc := service.NewEmailService(emailViewRepo)
	msgSvc := service.NewMessageService(msgViewRepo)
	return service.NewSyncService(userSvc, emailSvc, msgSvc)
}

func TestDispatchEvent_UserCreated(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	userViewRepo.On("CreateUserView", mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "user_created",
		Payload:   map[string]any{"ID": 1, "Email": "alice@example.com"},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	userViewRepo.AssertCalled(t, "CreateUserView", mock.Anything, mock.Anything)
	emailViewRepo.AssertNotCalled(t, "AddEmailToUser")
	msgViewRepo.AssertNotCalled(t, "AddMessageToUser")
}

func TestDispatchEvent_UserDeleted(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	userViewRepo.On("DeleteUserView", mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "user_deleted",
		Payload:   map[string]any{"ID": 1},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	userViewRepo.AssertCalled(t, "DeleteUserView", mock.Anything, mock.Anything)
}

func TestDispatchEvent_MessageCreated(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	msgViewRepo.On("AddMessageToUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "message_created",
		Payload:   map[string]any{"MessageID": 10, "UserID": 1, "Subject": "Hi", "Content": "Body"},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	msgViewRepo.AssertCalled(t, "AddMessageToUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_MessageDeleted(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	msgViewRepo.On("RemoveMessageFromUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "message_deleted",
		Payload:   map[string]any{"MessageID": 10, "UserID": 1},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	msgViewRepo.AssertCalled(t, "RemoveMessageFromUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_EmailAdded(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	emailViewRepo.On("AddEmailToUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "email_added",
		Payload:   map[string]any{"UserID": 1, "Address": "x@example.com", "Category": "work", "Importance": 5},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	emailViewRepo.AssertCalled(t, "AddEmailToUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_EmailUpdated(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	emailViewRepo.On("UpdateEmailForUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "email_updated",
		Payload: map[string]any{
			"UserID": 1, "Address": "x@example.com",
			"NewCategory": "vip", "NewImportance": 10,
		},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	emailViewRepo.AssertCalled(t, "UpdateEmailForUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_EmailRemoved(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	emailViewRepo.On("RemoveEmailFromUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "email_removed",
		Payload:   map[string]any{"UserID": 1, "Address": "x@example.com"},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	emailViewRepo.AssertCalled(t, "RemoveEmailFromUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_UnknownType(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	emailViewRepo := new(mocks.MockEmailViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	syncSvc := syncServiceWith(userViewRepo, emailViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{EventType: "unknown_xyz", Payload: nil}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	assert.NoError(t, err)
	userViewRepo.AssertNotCalled(t, "CreateUserView")
	userViewRepo.AssertNotCalled(t, "DeleteUserView")
	emailViewRepo.AssertNotCalled(t, "AddEmailToUser")
	msgViewRepo.AssertNotCalled(t, "AddMessageToUser")
}
