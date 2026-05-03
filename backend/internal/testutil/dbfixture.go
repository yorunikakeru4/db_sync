package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"backend/internal/bunmodel"
	"backend/internal/readmodel"

	"github.com/goccy/go-yaml"
	"github.com/uptrace/bun"
)

// SeedData is the backend integration fixture schema.
type SeedData struct {
	// Users seeds PostgreSQL users.
	Users []*bunmodel.User `yaml:"users"`
	// Messages seeds PostgreSQL messages.
	Messages []*bunmodel.Message `yaml:"messages"`
	// Contacts seeds PostgreSQL contacts.
	Contacts []*bunmodel.Contact `yaml:"contacts"`
	// UserContacts seeds PostgreSQL user-contact relations.
	UserContacts []*bunmodel.UserContact `yaml:"user_contacts"`
	// UserViews seeds MongoDB CQRS projections.
	UserViews []*readmodel.UserView `yaml:"user_views"`
}

// LoadFixtures resets PostgreSQL and MongoDB and seeds both stores from a YAML file.
func LoadFixtures(t *testing.T, db *TestDB, bunDB *bun.DB, path string) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}

	var seed SeedData
	if err := yaml.Unmarshal(body, &seed); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}

	db.Reset(t)
	ctx := context.Background()
	if err := seedPostgres(ctx, bunDB, &seed); err != nil {
		t.Fatalf("seed postgres: %v", err)
	}
	if err := seedMongo(ctx, db, &seed); err != nil {
		t.Fatalf("seed mongo: %v", err)
	}
}

func seedPostgres(ctx context.Context, db *bun.DB, seed *SeedData) error {
	for _, user := range seed.Users {
		if _, err := db.ExecContext(ctx, "INSERT INTO users (id, email, created_at) VALUES (?, ?, ?)", user.ID, user.Email, user.CreatedAt); err != nil {
			return fmt.Errorf("insert user %d: %w", user.ID, err)
		}
	}
	for _, message := range seed.Messages {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO messages (id, external_id, sender_id, receiver_id, subject, text, date_sent, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			message.ID, message.ExternalID, message.SenderID, message.ReceiverID, message.Subject, message.Text, message.DateSent, message.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert message %d: %w", message.ID, err)
		}
	}
	for _, contact := range seed.Contacts {
		if _, err := db.ExecContext(ctx, "INSERT INTO contacts (id, contact_value, created_at) VALUES (?, ?, ?)", contact.ID, contact.Value, contact.CreatedAt); err != nil {
			return fmt.Errorf("insert contact %d: %w", contact.ID, err)
		}
	}
	for _, item := range seed.UserContacts {
		if _, err := db.ExecContext(ctx, `
			INSERT INTO users_contacts (id, user_id, contact_id, importance, category, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			item.ID, item.UserID, item.ContactID, item.Importance, item.Category, item.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert user_contact %d: %w", item.ID, err)
		}
	}
	if _, err := db.ExecContext(ctx, "SELECT setval('users_id_seq', COALESCE((SELECT MAX(id) FROM users), 1), true)"); err != nil {
		return fmt.Errorf("set users sequence: %w", err)
	}
	if _, err := db.ExecContext(ctx, "SELECT setval('messages_id_seq', COALESCE((SELECT MAX(id) FROM messages), 1), true)"); err != nil {
		return fmt.Errorf("set messages sequence: %w", err)
	}
	if _, err := db.ExecContext(ctx, "SELECT setval('contacts_id_seq', COALESCE((SELECT MAX(id) FROM contacts), 1), true)"); err != nil {
		return fmt.Errorf("set contacts sequence: %w", err)
	}
	if _, err := db.ExecContext(ctx, "SELECT setval('users_contacts_id_seq', COALESCE((SELECT MAX(id) FROM users_contacts), 1), true)"); err != nil {
		return fmt.Errorf("set users_contacts sequence: %w", err)
	}
	return nil
}

func seedMongo(ctx context.Context, db *TestDB, seed *SeedData) error {
	if len(seed.UserViews) == 0 {
		return nil
	}

	docs := make([]any, 0, len(seed.UserViews))
	for _, user := range seed.UserViews {
		docs = append(docs, user)
	}
	if _, err := db.Mongo.Collection("users").InsertMany(ctx, docs); err != nil {
		return fmt.Errorf("insert mongo users: %w", err)
	}
	return nil
}
