package kafka

import (
	"context"
	"testing"
	"time"

	"db_sync/internal/application/events"
	"db_sync/internal/config"
	"db_sync/internal/service"
	"db_sync/internal/storage/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// syncServiceWith builds a SyncService backed by the provided mock repositories.
func syncServiceWith(
	userViewRepo *mocks.MockUserViewRepository,
	contactViewRepo *mocks.MockContactViewRepository,
	msgViewRepo *mocks.MockMessageViewRepository,
) *service.SyncService {
	userRepo := new(mocks.MockUserRepository)
	userSvc := service.NewUserService(userRepo, userViewRepo)
	contactSvc := service.NewContactService(contactViewRepo)
	msgSvc := service.NewMessageService(msgViewRepo)
	return service.NewSyncService(userSvc, contactSvc, msgSvc)
}

func TestDispatchEvent_UserCreated(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	userViewRepo.On("CreateUserView", mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "user_created",
		Payload:   map[string]any{"id": 1, "email": "alice@example.com"},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	userViewRepo.AssertCalled(t, "CreateUserView", mock.Anything, mock.Anything)
	contactViewRepo.AssertNotCalled(t, "AddContactToUser")
	msgViewRepo.AssertNotCalled(t, "AddMessageToUser")
}

func TestDispatchEvent_UserCreated_StringTimestamp(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	userViewRepo.On("CreateUserView", mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "user_created",
		Payload: map[string]any{
			"id":         1,
			"email":      "alice@example.com",
			"created_at": "2026-05-03T18:00:00Z",
		},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	userViewRepo.AssertCalled(t, "CreateUserView", mock.Anything, mock.Anything)
}

func TestDispatchEvent_UserDeleted(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	userViewRepo.On("DeleteUserView", mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "user_deleted",
		Payload:   map[string]any{"id": 1},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	userViewRepo.AssertCalled(t, "DeleteUserView", mock.Anything, mock.Anything)
}

func TestDispatchEvent_MessageCreated(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	msgViewRepo.On("AddMessageToUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "message_created",
		Payload: map[string]any{
			"message_id": 10,
			"user_id":    1,
			"subject":    "Hi",
			"content":    "Body",
			"date_sent":  time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	msgViewRepo.AssertCalled(t, "AddMessageToUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_MessageDeleted(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	msgViewRepo.On("RemoveMessageFromUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "message_deleted",
		Payload:   map[string]any{"message_id": 10, "user_id": 1},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	msgViewRepo.AssertCalled(t, "RemoveMessageFromUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_ContactAdded(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	contactViewRepo.On("AddContactToUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "contact_added",
		Payload: map[string]any{
			"user_id":    1,
			"value":      "x@example.com",
			"category":   "work",
			"importance": 5,
		},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	contactViewRepo.AssertCalled(t, "AddContactToUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_ContactUpdated(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	contactViewRepo.On("UpdateContactForUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "contact_updated",
		Payload: map[string]any{
			"user_id":        1,
			"value":          "x@example.com",
			"old_value":      "old@example.com",
			"new_category":   "vip",
			"new_importance": 10,
		},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	contactViewRepo.AssertCalled(t, "UpdateContactForUser", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_ContactRemoved(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	contactViewRepo.On("RemoveContactFromUser", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{
		EventType: "contact_removed",
		Payload:   map[string]any{"user_id": 1, "value": "x@example.com"},
	}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	require.NoError(t, err)
	contactViewRepo.AssertCalled(t, "RemoveContactFromUser", mock.Anything, mock.Anything, mock.Anything)
}

func TestDispatchEvent_UnknownType(t *testing.T) {
	userViewRepo := new(mocks.MockUserViewRepository)
	contactViewRepo := new(mocks.MockContactViewRepository)
	msgViewRepo := new(mocks.MockMessageViewRepository)

	syncSvc := syncServiceWith(userViewRepo, contactViewRepo, msgViewRepo)
	consumer := &KafkaConsumer{}

	event := &events.Event{EventType: "unknown_xyz", Payload: nil}
	err := consumer.DispatchEvent(context.Background(), event, syncSvc)

	assert.NoError(t, err)
	userViewRepo.AssertNotCalled(t, "CreateUserView")
	userViewRepo.AssertNotCalled(t, "DeleteUserView")
	contactViewRepo.AssertNotCalled(t, "AddContactToUser")
	msgViewRepo.AssertNotCalled(t, "AddMessageToUser")
}

func TestConsumerConfig_UsesConsumerGroup(t *testing.T) {
	oldCfg := config.KafkaConf
	t.Cleanup(func() {
		config.KafkaConf = oldCfg
	})

	config.KafkaConf = config.KafkaConfig{
		Topic:   "sync_topic",
		Host:    "kafka",
		Port:    "9092",
		GroupID: "db-sync",
	}

	cfg := consumerConfig()

	assert.Equal(t, []string{"kafka:9092"}, cfg.Brokers)
	assert.Equal(t, "sync_topic", cfg.Topic)
	assert.Equal(t, "db-sync", cfg.GroupID)
	assert.Zero(t, cfg.Partition)
}
