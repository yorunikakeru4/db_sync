//go:build integration

package storage

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"db_sync/internal/domain"
	"db_sync/internal/models"
	"db_sync/internal/testutil"

	"github.com/cockroachdb/datadriven"
	"github.com/goccy/go-yaml"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// TestStorageDatadriven verifies PostgreSQL storage reads against fixture-driven cases.
func TestStorageDatadriven(t *testing.T) {
	pg := testutil.NewTestDB(t)
	bundb := bun.NewDB(pg.PG.DB, pgdialect.New())
	t.Cleanup(func() {
		_ = bundb.Close()
	})

	runner := &storageRunner{
		users:    NewPostgresUserRepository(pg.PG),
		messages: NewPostgresMessageRepository(pg.PG),
		contacts: NewPostgresContactRepository(pg.PG),
		bundb:    bundb,
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if strings.HasSuffix(path, ".yml") {
			t.Skip()
		}
		datadriven.RunTest(t, path, runner.run)
	})
}

type storageRunner struct {
	users    *PostgresUserRepository
	messages *PostgresMessageRepository
	contacts *PostgresContactRepository
	bundb    *bun.DB
}

func (r *storageRunner) run(t *testing.T, tc *datadriven.TestCase) string {
	t.Helper()

	switch tc.Name {
	case "load-fixture":
		var name string
		if err := tc.NamedArg("fixture").Scan(&name); err != nil {
			tc.Fatalf(t, "scan fixture name: %v", err)
		}
		testutil.LoadFixtures(t, r.bundb, fixturePath(name))
		return "ok"
	case "get-user":
		id := scanIntArg(t, tc, "id")
		user, err := r.users.GetUserByID(id)
		if err != nil {
			tc.Fatalf(t, "get user: %v", err)
		}
		return mustYAML(t, userOutput{
			ID:              user.ID,
			Email:           user.Email,
			ImportantContacts: user.ImportantContacts,
			Messages:        user.Messages,
			CreatedAt:       formatTime(user.CreatedAt),
		})
	case "get-message":
		id := scanIntArg(t, tc, "id")
		msg, err := r.messages.GetMessageByID(id)
		if err != nil {
			tc.Fatalf(t, "get message: %v", err)
		}
		return mustYAML(t, renderMessage(msg))
	case "list-messages":
		userID := scanIntArg(t, tc, "user_id")
		msgs, err := r.messages.GetMessagesByUserID(userID)
		if err != nil {
			tc.Fatalf(t, "list messages: %v", err)
		}
		items := make([]messageOutput, 0, len(msgs))
		for i := range msgs {
			items = append(items, renderMessage(&msgs[i]))
		}
		return mustYAML(t, items)
	case "get-contact":
		id := scanIntArg(t, tc, "id")
		contact, err := r.contacts.GetContactByID(id)
		if err != nil {
			tc.Fatalf(t, "get contact: %v", err)
		}
		return mustYAML(t, renderContact(contact))
	case "list-user-contacts":
		userID := scanIntArg(t, tc, "user_id")
		items, err := r.contacts.GetContactsByUserID(userID)
		if err != nil {
			tc.Fatalf(t, "list user contacts: %v", err)
		}
		result := make([]userContactOutput, 0, len(items))
		for i := range items {
			result = append(result, renderUserContact(&items[i]))
		}
		return mustYAML(t, result)
	default:
		tc.Fatalf(t, "unknown command: %s", tc.Name)
		return ""
	}
}

func scanIntArg(t *testing.T, tc *datadriven.TestCase, key string) int {
	t.Helper()

	var value int
	if err := tc.NamedArg(key).Scan(&value); err != nil {
		tc.Fatalf(t, "scan %s: %v", key, err)
	}
	return value
}

func mustYAML(t *testing.T, value any) string {
	t.Helper()

	body, err := yaml.Marshal(value)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	return string(body)
}

func fixturePath(name string) string {
	return filepath.Join("testdata", "fixtures", name+".yml")
}

func renderMessage(msg *domain.Message) messageOutput {
	return messageOutput{
		ID:         msg.ID,
		ExternalID: msg.ExternalID,
		SenderID:   msg.SenderID,
		ReceiverID: msg.ReceiverID,
		Subject:    msg.Subject,
		Text:       msg.Text,
		DateSent:   formatTime(msg.DateSent),
		CreatedAt:  formatTime(msg.CreatedAt),
	}
}

func renderContact(contact *domain.Contact) contactOutput {
	return contactOutput{
		ID:        contact.ID,
		Address:   contact.Value,
		CreatedAt: formatTime(contact.CreatedAt),
	}
}

func renderUserContact(item *domain.UserContact) userContactOutput {
	return userContactOutput{
		ID:         item.ID,
		UserID:     item.UserID,
		ContactID:  item.ContactID,
		Importance: item.Importance,
		Category:   item.Category,
		CreatedAt:  formatTime(item.CreatedAt),
	}
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

type userOutput struct {
	ID                int                `yaml:"id"`
	Email             string             `yaml:"email"`
	ImportantContacts []models.Contact   `yaml:"important_contacts"`
	Messages          []models.Message   `yaml:"messages"`
	CreatedAt         string             `yaml:"created_at"`
}

type messageOutput struct {
	ID         int    `yaml:"id"`
	ExternalID string `yaml:"external_id"`
	SenderID   int    `yaml:"sender_id"`
	ReceiverID int    `yaml:"receiver_id"`
	Subject    string `yaml:"subject"`
	Text       string `yaml:"text"`
	DateSent   string `yaml:"date_sent"`
	CreatedAt  string `yaml:"created_at"`
}

type contactOutput struct {
	ID        int    `yaml:"id"`
	Address   string `yaml:"address"`
	CreatedAt string `yaml:"created_at"`
}

type userContactOutput struct {
	ID         int    `yaml:"id"`
	UserID     int    `yaml:"user_id"`
	ContactID  int    `yaml:"contact_id"`
	Importance int    `yaml:"importance"`
	Category   int    `yaml:"category"`
	CreatedAt  string `yaml:"created_at"`
}
