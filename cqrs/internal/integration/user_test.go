//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"db_sync/internal/application/events"
	"db_sync/internal/service"
	"db_sync/internal/storage"
	"db_sync/internal/testutil"
	kafkapkg "db_sync/internal/transport/kafka"
	"db_sync/internal/view"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestIntegration_UserCreated(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	createdAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Insert source record into PostgreSQL (the write model).
	_, err := db.PG.ExecContext(ctx,
		`INSERT INTO users (id, email, created_at) VALUES ($1, $2, $3)`,
		1, "alice@example.com", createdAt,
	)
	require.NoError(t, err)

	// Fire the event through the full dispatch pipeline.
	syncSvc := buildSyncSvc(db)
	event := &events.Event{
		EventType: "user_created",
		Payload: map[string]any{
			"id":         1,
			"email":      "alice@example.com",
			"created_at": createdAt,
		},
	}
	err = (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, event, syncSvc)
	require.NoError(t, err)

	// Assert the view document was written to MongoDB.
	var result view.UserView
	err = db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": 1}).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, 1, result.ID)
	assert.Equal(t, "alice@example.com", result.Email)
	assert.WithinDuration(t, createdAt, result.CreatedAt, time.Millisecond)
}

func TestIntegration_UserDeleted(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	// Seed a user view in MongoDB.
	userViewRepo := storage.NewMongoUserViewRepository(db.Mongo)
	require.NoError(t, userViewRepo.CreateUserView(ctx, view.UserView{
		ID:        2,
		Email:     "bob@example.com",
		CreatedAt: time.Now(),
	}))

	// Fire user_deleted event.
	event := &events.Event{
		EventType: "user_deleted",
		Payload:   map[string]any{"id": 2},
	}
	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, event, buildSyncSvc(db))
	require.NoError(t, err)

	// Assert document is gone from MongoDB.
	count, err := db.Mongo.Collection("users").CountDocuments(ctx, bson.M{"id": 2})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// buildSyncSvc assembles a SyncService backed by the live test databases.
func buildSyncSvc(db *testutil.TestDB) *service.SyncService {
	return service.NewSyncService(
		service.NewUserService(
			storage.NewPostgresUserRepository(db.PG),
			storage.NewMongoUserViewRepository(db.Mongo),
		),
		service.NewContactService(storage.NewMongoContactViewRepository(db.Mongo)),
		service.NewMessageService(storage.NewMongoMessageViewRepository(db.Mongo)),
	)
}

// seedUser inserts a bare user view into MongoDB so email/message tests have
// a document to update.
func seedUser(t *testing.T, db *testutil.TestDB, id int, email string) {
	t.Helper()
	repo := storage.NewMongoUserViewRepository(db.Mongo)
	require.NoError(t, repo.CreateUserView(context.Background(), view.UserView{
		ID:                id,
		Email:             email,
		CreatedAt:         time.Now(),
		ImportantContacts: []view.ImportantContactView{},
		Messages:          []view.MessageView{},
	}))
}
