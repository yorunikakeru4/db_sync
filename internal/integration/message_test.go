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
