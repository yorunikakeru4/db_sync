// Package testutil provides PostgreSQL-only fixture helpers for CQRS tests.
package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"db_sync/internal/bunmodel"

	"github.com/goccy/go-yaml"
	"github.com/uptrace/bun"
)

// SeedData is the YAML fixture schema for CQRS PostgreSQL integration tests.
type SeedData struct {
	// Users are inserted into the users table first.
	Users []*bunmodel.User `yaml:"users"`
	// Messages are inserted after users.
	Messages []*bunmodel.Message `yaml:"messages"`
	// Contacts are inserted before user-contact links.
	Contacts []*bunmodel.Contact `yaml:"contacts"`
	// UserContacts are inserted last.
	UserContacts []*bunmodel.UserContact `yaml:"user_contacts"`
}

// LoadFixtures resets the CQRS PostgreSQL tables and inserts fixture rows from a YAML file.
func LoadFixtures(t *testing.T, db *bun.DB, path string) {
	t.Helper()

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}

	var seed SeedData
	if err := yaml.Unmarshal(body, &seed); err != nil {
		t.Fatalf("decode fixture %s: %v", path, err)
	}

	ctx := context.Background()
	if err := resetTables(ctx, db); err != nil {
		t.Fatalf("reset tables: %v", err)
	}
	if err := insertFixtures(ctx, db, &seed); err != nil {
		t.Fatalf("insert fixtures from %s: %v", path, err)
	}
}

func resetTables(ctx context.Context, db *bun.DB) error {
	_, err := db.ExecContext(
		ctx,
		"TRUNCATE users_contacts, contacts, messages, users RESTART IDENTITY CASCADE",
	)
	return err
}

func insertFixtures(ctx context.Context, db *bun.DB, seed *SeedData) error {
	if err := insertModels(ctx, db, seed.Users); err != nil {
		return fmt.Errorf("insert users: %w", err)
	}
	if err := ensureUserIDs(ctx, db, seed.Users); err != nil {
		return err
	}
	if err := insertModels(ctx, db, seed.Messages); err != nil {
		return fmt.Errorf("insert messages: %w", err)
	}
	if err := insertModels(ctx, db, seed.Contacts); err != nil {
		return fmt.Errorf("insert contacts: %w", err)
	}
	if err := insertModels(ctx, db, seed.UserContacts); err != nil {
		return fmt.Errorf("insert user_contacts: %w", err)
	}
	return nil
}

func ensureUserIDs(ctx context.Context, db *bun.DB, users []*bunmodel.User) error {
	if len(users) == 0 {
		return nil
	}

	var ids []int64
	if err := db.NewSelect().TableExpr("users").Column("id").OrderExpr("id ASC").Scan(ctx, &ids); err != nil {
		return fmt.Errorf("select seeded user ids: %w", err)
	}

	if len(ids) != len(users) {
		return fmt.Errorf("seeded user count mismatch: have %v want %d", ids, len(users))
	}
	for i, user := range users {
		if ids[i] != user.ID {
			return fmt.Errorf("seeded user ids mismatch: have %v want user id %d at pos %d", ids, user.ID, i)
		}
	}

	return nil
}

func insertModels[T any](ctx context.Context, db *bun.DB, models []*T) error {
	if len(models) == 0 {
		return nil
	}

	for _, model := range models {
		switch item := any(model).(type) {
		case *bunmodel.User:
			if _, err := db.ExecContext(
				ctx,
				"INSERT INTO users (id, email, created_at) VALUES (?, ?, ?)",
				item.ID, item.Email, item.CreatedAt,
			); err != nil {
				return err
			}
		case *bunmodel.Message:
			if _, err := db.ExecContext(
				ctx,
				`INSERT INTO messages (
					id, external_id, sender_id, receiver_id, subject, text, date_sent, created_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				item.ID, item.ExternalID, item.SenderID, item.ReceiverID,
				item.Subject, item.Text, item.DateSent, item.CreatedAt,
			); err != nil {
				return err
			}
		case *bunmodel.Contact:
			if _, err := db.ExecContext(
				ctx,
				"INSERT INTO contacts (id, contact_value, created_at) VALUES (?, ?, ?)",
				item.ID, item.Value, item.CreatedAt,
			); err != nil {
				return err
			}
		case *bunmodel.UserContact:
			if _, err := db.ExecContext(
				ctx,
				`INSERT INTO users_contacts (
					id, user_id, contact_id, importance, category, created_at
				) VALUES (?, ?, ?, ?, ?, ?)`,
				item.ID, item.UserID, item.ContactID, item.Importance, item.Category, item.CreatedAt,
			); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported fixture model type %T", item)
		}
	}

	return nil
}

// FixturePath resolves a testdata fixture path relative to the caller package.
func FixturePath(parts ...string) string {
	return filepath.Join(parts...)
}
