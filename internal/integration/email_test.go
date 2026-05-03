//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"

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

func TestIntegration_EmailAdded(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice@example.com")

	event := &events.Event{
		EventType: "email_added",
		Payload: map[string]any{
			"user_id":    1,
			"address":    "contact@work.com",
			"category":   "work",
			"importance": 5,
		},
	}
	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, event, buildSyncSvc(db))
	require.NoError(t, err)

	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result))

	require.Len(t, result.ImportantContacts, 1)
	c := result.ImportantContacts[0]
	assert.Equal(t, "contact@work.com", c.Email)
	assert.Equal(t, "work", c.Category)
	assert.Equal(t, 5, c.Importance)
}

func TestIntegration_EmailUpdated(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice@example.com")

	// Seed an existing contact.
	emailRepo := storage.NewMongoEmailViewRepository(db.Mongo)
	require.NoError(t, emailRepo.AddEmailToUser(ctx, 1, view.ImportantContactView{
		Email: "contact@work.com", Category: "personal", Importance: 1,
	}))

	event := &events.Event{
		EventType: "email_updated",
		Payload: map[string]any{
			"user_id":        1,
			"address":        "contact@work.com",
			"old_category":   "personal",
			"new_category":   "vip",
			"old_importance": 1,
			"new_importance": 10,
		},
	}
	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, event, buildSyncSvc(db))
	require.NoError(t, err)

	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result))

	require.Len(t, result.ImportantContacts, 1)
	c := result.ImportantContacts[0]
	assert.Equal(t, "vip", c.Category)
	assert.Equal(t, 10, c.Importance)
}

func TestIntegration_EmailRemoved(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()
	seedUser(t, db, 1, "alice@example.com")

	// Seed a contact to be removed.
	emailRepo := storage.NewMongoEmailViewRepository(db.Mongo)
	require.NoError(t, emailRepo.AddEmailToUser(ctx, 1, view.ImportantContactView{
		Email: "contact@work.com", Category: "work", Importance: 3,
	}))

	event := &events.Event{
		EventType: "email_removed",
		Payload: map[string]any{
			"user_id": 1,
			"address": "contact@work.com",
		},
	}
	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, event, buildSyncSvc(db))
	require.NoError(t, err)

	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result))
	assert.Empty(t, result.ImportantContacts)
}

func TestIntegration_EmailAdded_UserNotFound(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, &events.Event{
		EventType: "email_added",
		Payload: map[string]any{
			"user_id":    999,
			"address":    "ghost@example.com",
			"category":   "work",
			"importance": 1,
		},
	}, buildSyncSvc(db))

	require.Error(t, err)
	assert.True(t, errors.Is(err, mongo.ErrNoDocuments))
}
