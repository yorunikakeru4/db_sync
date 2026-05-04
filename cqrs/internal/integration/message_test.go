//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"db_sync/internal/application/events"
	"db_sync/internal/storage"
	"db_sync/internal/testutil"
	kafkapkg "db_sync/internal/transport/kafka"
	"db_sync/internal/view"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func TestIntegration_MessageAdded(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice@example.com")

	sentAt := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	event := &events.Event{
		EventType: "message_created",
		Payload: map[string]any{
			"message_id": 10,
			"user_id":    1,
			"subject":    "Hello",
			"content":    "World",
			"date_sent":  sentAt,
		},
	}
	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, event, buildSyncSvc(db))
	require.NoError(t, err)

	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result))

	require.Len(t, result.Messages, 1)
	m := result.Messages[0]
	assert.Equal(t, 10, m.ID)
	assert.Equal(t, "Hello", m.Subject)
	assert.Equal(t, "World", m.Text)
	assert.Equal(t, 1, result.NumMessages)
	assert.WithinDuration(t, sentAt, m.CreatedAt, time.Millisecond)
}

func TestIntegration_MessageDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice@example.com")

	// Seed a message to be deleted.
	msgRepo := storage.NewMongoMessageViewRepository(db.Mongo)
	require.NoError(t, msgRepo.AddMessageToUser(ctx, 1, view.MessageView{
		ID: 10, Subject: "Hello", Text: "World", CreatedAt: time.Now(),
	}))

	event := &events.Event{
		EventType: "message_deleted",
		Payload:   map[string]any{"message_id": 10, "user_id": 1},
	}
	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, event, buildSyncSvc(db))
	require.NoError(t, err)

	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result))
	assert.Empty(t, result.Messages)
	assert.Zero(t, result.NumMessages)
}

func TestMongoMessageViewRepository_UpsertMessageToUser_ReplacesExistingMessage(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	userRepo := storage.NewMongoUserViewRepository(db.Mongo)
	msgRepo := storage.NewMongoMessageViewRepository(db.Mongo)

	require.NoError(t, userRepo.CreateUserView(ctx, view.UserView{
		ID:          1,
		Email:       "user@example.com",
		NumMessages: 1,
		Messages: []view.MessageView{{
			ID:        10,
			Subject:   "Old subject",
			Text:      "Old text",
			CreatedAt: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
		}},
	}))

	require.NoError(t, msgRepo.UpsertMessageToUser(ctx, 1, view.MessageView{
		ID:        10,
		Subject:   "New subject",
		Text:      "New text",
		CreatedAt: time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC),
	}))

	var got view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&got))
	require.Len(t, got.Messages, 1)
	assert.Equal(t, 1, got.NumMessages)
	assert.Equal(t, "New subject", got.Messages[0].Subject)
	assert.Equal(t, "New text", got.Messages[0].Text)
	assert.WithinDuration(t, time.Date(2024, 6, 2, 12, 0, 0, 0, time.UTC), got.Messages[0].CreatedAt, time.Millisecond)
}

func TestIntegration_MessageAdded_UserNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, &events.Event{
		EventType: "message_created",
		Payload: map[string]any{
			"message_id": 10,
			"user_id":    999,
			"subject":    "Hello",
			"content":    "World",
			"date_sent":  time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
		},
	}, buildSyncSvc(db))

	require.Error(t, err)
	assert.True(t, errors.Is(err, mongo.ErrNoDocuments))
}

func TestIntegration_MessageDeleted_MessageNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice@example.com")

	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, &events.Event{
		EventType: "message_deleted",
		Payload:   map[string]any{"message_id": 404, "user_id": 1},
	}, buildSyncSvc(db))
	require.NoError(t, err)

	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result))
	assert.Empty(t, result.Messages)
	assert.Zero(t, result.NumMessages)
}

func TestIntegration_MessageCreated_RepeatsUpdateEmbeddedMessage(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice@example.com")

	consumer := &kafkapkg.KafkaConsumer{}
	syncSvc := buildSyncSvc(db)

	require.NoError(t, consumer.DispatchEvent(ctx, &events.Event{
		EventType: "message_created",
		Payload: map[string]any{
			"message_id": 10,
			"user_id":    1,
			"subject":    "Old",
			"content":    "First",
			"date_sent":  "2026-05-04T10:00:00Z",
		},
	}, syncSvc))

	require.NoError(t, consumer.DispatchEvent(ctx, &events.Event{
		EventType: "message_created",
		Payload: map[string]any{
			"message_id": 10,
			"user_id":    1,
			"subject":    "New",
			"content":    "Second",
			"date_sent":  "2026-05-04T11:00:00Z",
		},
	}, syncSvc))

	var got view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&got))
	require.Len(t, got.Messages, 1)
	require.Equal(t, 1, got.NumMessages)
	require.Equal(t, "New", got.Messages[0].Subject)
	require.Equal(t, "Second", got.Messages[0].Text)
	require.WithinDuration(t, time.Date(2026, 5, 4, 11, 0, 0, 0, time.UTC), got.Messages[0].CreatedAt, time.Millisecond)
}
