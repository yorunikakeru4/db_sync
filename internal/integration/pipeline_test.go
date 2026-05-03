//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"db_sync/internal/application/events"
	"db_sync/internal/testutil"
	kafkapkg "db_sync/internal/transport/kafka"
	"db_sync/internal/view"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// TestIntegration_FullPipeline fires user_created → email_added → message_added
// in sequence and verifies the entire MongoDB document at the end.
func TestIntegration_FullPipeline(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	consumer := &kafkapkg.KafkaConsumer{}
	syncSvc := buildSyncSvc(db)
	sentAt := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	// Step 1: create user.
	err := consumer.DispatchEvent(ctx, &events.Event{
		EventType: "user_created",
		Payload:   map[string]any{"ID": 1, "Email": "alice@example.com"},
	}, syncSvc)
	require.NoError(t, err)

	// Step 2: add an important contact.
	err = consumer.DispatchEvent(ctx, &events.Event{
		EventType: "email_added",
		Payload: map[string]any{
			"UserID": 1, "Address": "boss@work.com",
			"Category": "vip", "Importance": 10,
		},
	}, syncSvc)
	require.NoError(t, err)

	// Step 3: add a message.
	err = consumer.DispatchEvent(ctx, &events.Event{
		EventType: "message_created",
		Payload: map[string]any{
			"MessageID": 42, "UserID": 1,
			"Subject": "Important", "Content": "Read me",
			"DateSent": sentAt,
		},
	}, syncSvc)
	require.NoError(t, err)

	// Assert the entire MongoDB document reflects all three events.
	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result))

	assert.Equal(t, 1, result.ID)
	assert.Equal(t, "alice@example.com", result.Email)

	require.Len(t, result.ImportantContacts, 1)
	c := result.ImportantContacts[0]
	assert.Equal(t, "boss@work.com", c.Email)
	assert.Equal(t, "vip", c.Category)
	assert.Equal(t, 10, c.Importance)

	require.Len(t, result.Messages, 1)
	m := result.Messages[0]
	assert.Equal(t, 42, m.ID)
	assert.Equal(t, "Important", m.Subject)
	assert.Equal(t, "Read me", m.Text)
	assert.WithinDuration(t, sentAt, m.CreatedAt, time.Millisecond)
}

// TestIntegration_UserCreated_Idempotent verifies that firing user_created twice
// does not create duplicate documents (CreateUserView uses $setOnInsert).
func TestIntegration_UserCreated_Idempotent(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	event := &events.Event{
		EventType: "user_created",
		Payload:   map[string]any{"ID": 1, "Email": "alice@example.com"},
	}
	consumer := &kafkapkg.KafkaConsumer{}
	syncSvc := buildSyncSvc(db)

	require.NoError(t, consumer.DispatchEvent(ctx, event, syncSvc))
	require.NoError(t, consumer.DispatchEvent(ctx, event, syncSvc))

	count, err := db.Mongo.Collection("users").CountDocuments(ctx, bson.M{"id": 1})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count, "duplicate user_created must not create two documents")
}

// TestIntegration_UserDeleted_NonExistent verifies that deleting a user who does
// not exist in MongoDB returns no error.
func TestIntegration_UserDeleted_NonExistent(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, &events.Event{
		EventType: "user_deleted",
		Payload:   map[string]any{"ID": 999},
	}, buildSyncSvc(db))

	assert.NoError(t, err)
}
