//go:build integration

// Package integration contains end-to-end tests that verify the full CQRS pipeline:
// data is written to PostgreSQL (write model), an event is constructed from that data,
// dispatched through DispatchEvent, and the MongoDB read model is asserted to reflect
// exactly the same values as the PostgreSQL source.
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

// TestIntegration_EmailAdded_FromPostgres inserts a user and email record into
// PostgreSQL, reads them back, constructs the event from the real DB values, and
// asserts that MongoDB contains exactly those values.
func TestIntegration_EmailAdded_FromPostgres(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	// ── Write to PostgreSQL ────────────────────────────────────────────────────
	var userID int
	require.NoError(t, db.PG.QueryRowContext(ctx,
		`INSERT INTO users (email, created_at) VALUES ($1, NOW()) RETURNING id`,
		"alice@example.com",
	).Scan(&userID))

	var emailID int
	require.NoError(t, db.PG.QueryRowContext(ctx,
		`INSERT INTO emails (email_address, created_at) VALUES ($1, NOW()) RETURNING id`,
		"contact@work.com",
	).Scan(&emailID))

	require.NoError(t, db.PG.QueryRowContext(ctx,
		`INSERT INTO users_emails (user_id, email_id, importance, category, created_at)
		 VALUES ($1, $2, $3, $4, NOW()) RETURNING id`,
		userID, emailID, 5, 1,
	).Scan(new(int)))

	// ── Read back from PostgreSQL (verify source data is there) ───────────────
	var pgAddress string
	var pgImportance int
	require.NoError(t, db.PG.QueryRowContext(ctx,
		`SELECT e.email_address, ue.importance
		 FROM emails e
		 JOIN users_emails ue ON ue.email_id = e.id
		 WHERE ue.user_id = $1`, userID,
	).Scan(&pgAddress, &pgImportance))

	assert.Equal(t, "contact@work.com", pgAddress)
	assert.Equal(t, 5, pgImportance)

	// ── Construct event from PostgreSQL values and dispatch ───────────────────
	seedUser(t, db, userID, "alice@example.com")

	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, &events.Event{
		EventType: "email_added",
		Payload: map[string]any{
			"user_id":    userID,
			"address":    pgAddress,    // from PG
			"importance": pgImportance, // from PG
			"category":   "work",
		},
	}, buildSyncSvc(db))
	require.NoError(t, err)

	// ── Assert MongoDB matches PostgreSQL source ───────────────────────────────
	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": userID}).Decode(&result))

	require.Len(t, result.ImportantContacts, 1)
	c := result.ImportantContacts[0]
	assert.Equal(t, pgAddress, c.Email)       // Mongo == Postgres
	assert.Equal(t, pgImportance, c.Importance) // Mongo == Postgres
	assert.Equal(t, "work", c.Category)
}

// TestIntegration_MessageAdded_FromPostgres inserts a message into PostgreSQL,
// reads it back, constructs the event from the real DB values, and asserts that
// MongoDB contains exactly those values including the date.
func TestIntegration_MessageAdded_FromPostgres(t *testing.T) {
	db := testutil.NewTestDB(t)
	db.Reset(t)
	ctx := context.Background()

	// ── Write to PostgreSQL ────────────────────────────────────────────────────
	var senderID, receiverID int
	require.NoError(t, db.PG.QueryRowContext(ctx,
		`INSERT INTO users (email, created_at) VALUES ('sender@example.com', NOW()) RETURNING id`,
	).Scan(&senderID))
	require.NoError(t, db.PG.QueryRowContext(ctx,
		`INSERT INTO users (email, created_at) VALUES ('receiver@example.com', NOW()) RETURNING id`,
	).Scan(&receiverID))

	dateSent := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	var msgID int
	require.NoError(t, db.PG.QueryRowContext(ctx,
		`INSERT INTO messages (external_id, sender_id, receiver_id, subject, text, date_sent, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING id`,
		"ext-001", senderID, receiverID, "Hello from PG", "Body from PG", dateSent,
	).Scan(&msgID))

	// ── Read back from PostgreSQL ─────────────────────────────────────────────
	var pgSubject, pgText string
	var pgDateSent time.Time
	require.NoError(t, db.PG.QueryRowContext(ctx,
		`SELECT subject, text, date_sent FROM messages WHERE id = $1`, msgID,
	).Scan(&pgSubject, &pgText, &pgDateSent))

	assert.Equal(t, "Hello from PG", pgSubject)
	assert.Equal(t, "Body from PG", pgText)
	assert.WithinDuration(t, dateSent, pgDateSent, time.Millisecond)

	// ── Construct event from PostgreSQL values and dispatch ───────────────────
	seedUser(t, db, receiverID, "receiver@example.com")

	err := (&kafkapkg.KafkaConsumer{}).DispatchEvent(ctx, &events.Event{
		EventType: "message_created",
		Payload: map[string]any{
			"message_id": msgID,
			"user_id":    receiverID,
			"subject":    pgSubject,  // from PG
			"content":    pgText,     // from PG
			"date_sent":  pgDateSent, // from PG
		},
	}, buildSyncSvc(db))
	require.NoError(t, err)

	// ── Assert MongoDB matches PostgreSQL source ───────────────────────────────
	var result view.UserView
	require.NoError(t, db.Mongo.Collection("users").FindOne(ctx, bson.M{"id": receiverID}).Decode(&result))

	require.Len(t, result.Messages, 1)
	m := result.Messages[0]
	assert.Equal(t, msgID, m.ID)
	assert.Equal(t, pgSubject, m.Subject)                              // Mongo == Postgres
	assert.Equal(t, pgText, m.Text)                                    // Mongo == Postgres
	assert.WithinDuration(t, pgDateSent, m.CreatedAt, time.Millisecond) // Mongo == Postgres
}
