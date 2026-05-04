//go:build integration

package http

import (
	"bytes"
	"context"
	"encoding/json"
	nethttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"backend/internal/eventworker"
	"backend/internal/readmodel"
	"backend/internal/storage"
	"backend/internal/testutil"
	"db_sync/projector"

	"github.com/cockroachdb/datadriven"
	"github.com/goccy/go-yaml"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

// TestHTTPDatadriven verifies backend HTTP handlers with real Postgres and MongoDB dependencies.
// Event publishing is now handled by PostgreSQL triggers → domain_events → worker.
func TestHTTPDatadriven(t *testing.T) {
	live := testutil.NewTestDB(t)
	bunDB := bun.NewDB(live.PG.DB, pgdialect.New())
	t.Cleanup(func() {
		bunDB.Close()
	})

	handler, err := NewHandler(HandlerParams{
		Users:     storage.NewUserRepo(bunDB),
		Messages:  storage.NewMessageRepo(bunDB),
		Contacts:  storage.NewContactRepo(bunDB),
		UserViews: storage.NewUserViewRepo(live.Mongo),
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}

	eventRepo := eventworker.NewPgRepository(bunDB)

	runner := &httpRunner{
		db:        live,
		bunDB:     bunDB,
		handler:   handler,
		eventRepo: eventRepo,
	}

	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		if strings.HasSuffix(path, ".yml") {
			t.Skip()
		}
		datadriven.RunTest(t, path, runner.run)
	})
}

type httpRunner struct {
	db        *testutil.TestDB
	bunDB     *bun.DB
	handler   *Handler
	eventRepo *eventworker.PgRepository
}

func (r *httpRunner) run(t *testing.T, tc *datadriven.TestCase) string {
	t.Helper()

	switch tc.Name {
	case "load-fixture":
		var name string
		if err := tc.NamedArg("fixture").Scan(&name); err != nil {
			tc.Fatalf(t, "scan fixture: %v", err)
		}
		testutil.LoadFixtures(t, r.db, r.bunDB, filepath.Join("testdata", "fixtures", name+".yml"))
		// Clear domain_events table for clean state
		_, _ = r.bunDB.NewDelete().Model((*eventworker.DomainEvent)(nil)).Where("1=1").Exec(context.Background())
		return "ok"
	case "http":
		var method, path string
		if err := tc.NamedArg("method").Scan(&method); err != nil {
			tc.Fatalf(t, "scan method: %v", err)
		}
		if err := tc.NamedArg("path").Scan(&path); err != nil {
			tc.Fatalf(t, "scan path: %v", err)
		}

		req := httptestNewRequest(method, path, tc.Input)
		rec := httptestNewRecorder()
		r.handler.ServeHTTP(rec, req)
		return renderHTTPResponse(t, method, path, rec.Code, rec.Body.Bytes())
	case "check-kafka":
		// Events are now in domain_events table, fetched by trigger
		var index int
		if err := tc.NamedArg("index").Scan(&index); err != nil {
			tc.Fatalf(t, "scan index: %v", err)
		}
		events, err := r.eventRepo.FetchUnpublished(context.Background(), 100)
		if err != nil {
			tc.Fatalf(t, "fetch domain events: %v", err)
		}
		if index < 0 || index >= len(events) {
			tc.Fatalf(t, "domain event index %d out of range; have %d", index, len(events))
		}
		return renderDomainEvent(t, events[index])
	case "project-kafka":
		// Project domain event to MongoDB
		var index int
		if err := tc.NamedArg("index").Scan(&index); err != nil {
			tc.Fatalf(t, "scan index: %v", err)
		}
		events, err := r.eventRepo.FetchUnpublished(context.Background(), 100)
		if err != nil {
			tc.Fatalf(t, "fetch domain events: %v", err)
		}
		if index < 0 || index >= len(events) {
			tc.Fatalf(t, "domain event index %d out of range; have %d", index, len(events))
		}
		evt := events[index]
		// Parse payload to proper structure
		var payload any
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			tc.Fatalf(t, "unmarshal payload: %v", err)
		}
		// Build the event envelope that projector expects
		envelope := map[string]any{
			"event_id":   "test-id",
			"event_type": evt.EventType,
			"version":    1,
			"timestamp":  evt.CreatedAt.Format(time.RFC3339),
			"payload":    payload,
		}
		body, err := json.Marshal(envelope)
		if err != nil {
			tc.Fatalf(t, "marshal domain event: %v", err)
		}
		if err := projector.ProjectJSONEvent(context.Background(), r.db.Mongo, body); err != nil {
			tc.Fatalf(t, "project domain event: %v", err)
		}
		return "ok"
	default:
		tc.Fatalf(t, "unknown command: %s", tc.Name)
		return ""
	}
}

func httptestNewRequest(method, path, body string) *nethttp.Request {
	req := httptest.NewRequestWithContext(context.Background(), method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-yaml")
	return req
}

func httptestNewRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func renderHTTPResponse(t *testing.T, method, path string, status int, body []byte) string {
	t.Helper()

	result := httpResponseOutput{Status: status}
	if len(bytes.TrimSpace(body)) == 0 {
		result.Body = nil
	} else {
		parsed, ok := decodeOrderedResponse(method, path, body)
		if ok {
			normalizeHTTPBody(method, path, parsed)
			result.Body = parsed
		} else {
			var generic any
			if err := yaml.Unmarshal(body, &generic); err != nil {
				result.Body = string(body)
			} else {
				normalizeHTTPBody(method, path, generic)
				result.Body = generic
			}
		}
	}
	return mustYAML(t, result)
}

func decodeOrderedResponse(method, path string, body []byte) (any, bool) {
	switch {
	case method == nethttp.MethodGet && strings.HasPrefix(path, "/users/"):
		value := new(userViewResponseOutput)
		if err := json.Unmarshal(body, value); err != nil {
			return nil, false
		}
		return value, true
	case method == nethttp.MethodPost && path == "/users":
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, false
		}
		return &userResponseOutput{
			ID:        int64Value(raw["ID"]),
			Email:     stringValue(raw["Email"]),
			CreatedAt: stringValue(raw["CreatedAt"]),
		}, true
	case method == nethttp.MethodPost && path == "/messages":
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, false
		}
		return &messageResponseOutput{
			ID:         int64Value(raw["ID"]),
			ExternalID: stringValue(raw["ExternalID"]),
			SenderID:   int64Value(raw["SenderID"]),
			ReceiverID: int64Value(raw["ReceiverID"]),
			Subject:    stringValue(raw["Subject"]),
			Text:       stringValue(raw["Text"]),
			DateSent:   stringValue(raw["DateSent"]),
			CreatedAt:  stringValue(raw["CreatedAt"]),
		}, true
	case (method == nethttp.MethodPost || method == nethttp.MethodPut) && strings.Contains(path, "/contacts"):
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err != nil {
			return nil, false
		}
		contact, _ := raw["contact"].(map[string]any)
		userContact, _ := raw["user_contact"].(map[string]any)
		return &contactOperationResponseOutput{
			Contact: contactDTO{
				ID:        int64Value(firstMapValue(contact, "ID", "id")),
				Value:     stringValue(firstMapValue(contact, "Value", "value")),
				CreatedAt: stringValue(firstMapValue(contact, "CreatedAt", "created_at")),
			},
			UserContact: userContactDTO{
				ID:         int64Value(firstMapValue(userContact, "ID", "id")),
				UserID:     int64Value(firstMapValue(userContact, "UserID", "user_id")),
				ContactID:  int64Value(firstMapValue(userContact, "ContactID", "contact_id")),
				Importance: intValue(firstMapValue(userContact, "Importance", "importance")),
				Category:   intValue(firstMapValue(userContact, "Category", "category")),
				CreatedAt:  stringValue(firstMapValue(userContact, "CreatedAt", "created_at")),
			},
		}, true
	default:
		return nil, false
	}
}

func renderDomainEvent(t *testing.T, event eventworker.DomainEvent) string {
	t.Helper()

	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal domain event payload: %v", err)
	}

	root := map[string]any{
		"event_id":   "",
		"event_type": event.EventType,
		"version":    1,
		"timestamp":  "ignore",
		"payload":    payload,
	}
	normalizeKafkaBody(root)

	return mustYAML(t, kafkaEventOutput{
		EventID:   "",
		EventType: event.EventType,
		Version:   1,
		Timestamp: "ignore",
		Payload:   orderedKafkaPayload(root),
	})
}

func mustYAML(t *testing.T, value any) string {
	t.Helper()

	body, err := yaml.Marshal(value)
	if err != nil {
		t.Fatalf("marshal yaml: %v", err)
	}
	return string(body)
}

func normalizeHTTPBody(method, path string, value any) {
	switch item := value.(type) {
	case *userResponseOutput:
		if method == nethttp.MethodPost && path == "/users" {
			item.CreatedAt = "ignore"
		}
		return
	case *messageResponseOutput:
		if method == nethttp.MethodPost && path == "/messages" {
			item.CreatedAt = "ignore"
			item.DateSent = normalizeRFC3339(item.DateSent)
		}
		return
	case *userViewResponseOutput:
		// Normalize created_at for dynamically created users (not fixture user 1)
		if method == nethttp.MethodGet && strings.HasPrefix(path, "/users/") && path != "/users/1" {
			item.CreatedAt = "ignore"
		}
		return
	case *contactOperationResponseOutput:
		item.Contact.CreatedAt = "ignore"
		item.UserContact.CreatedAt = "ignore"
		return
	}

	root, ok := value.(map[string]any)
	if !ok {
		return
	}

	if method == nethttp.MethodPut && strings.HasPrefix(path, "/users/") {
		root["CreatedAt"] = "ignore"
	}
	if method == nethttp.MethodPut && strings.HasPrefix(path, "/messages/") {
		root["CreatedAt"] = "ignore"
		if dateSent, ok := root["DateSent"].(string); ok {
			root["DateSent"] = normalizeRFC3339(dateSent)
		}
	}
	if (method == nethttp.MethodPost || method == nethttp.MethodPut) && strings.Contains(path, "/contacts") {
		if contact, ok := root["contact"].(map[string]any); ok {
			contact["CreatedAt"] = "ignore"
		}
		if userContact, ok := root["user_contact"].(map[string]any); ok {
			userContact["CreatedAt"] = "ignore"
		}
	}

	if method == nethttp.MethodPost && path == "/users" {
		if body, ok := root["body"].(map[string]any); ok {
			body["created_at"] = "ignore"
		}
	}
	if method == nethttp.MethodPost && path == "/messages" {
		if body, ok := root["body"].(map[string]any); ok {
			body["created_at"] = "ignore"
		}
	}
}

func normalizeKafkaBody(value any) {
	root, ok := value.(map[string]any)
	if !ok {
		return
	}
	root["timestamp"] = "ignore"

	payload, ok := root["payload"].(map[string]any)
	if !ok {
		return
	}

	switch root["event_type"] {
	case "user_created", "contact_added":
		payload["created_at"] = "ignore"
	case "contact_updated":
		payload["updated_at"] = "ignore"
	}
}

type httpResponseOutput struct {
	Status int `yaml:"status"`
	Body   any `yaml:"body"`
}

type kafkaEventOutput struct {
	EventID   string `yaml:"event_id"`
	EventType string `yaml:"event_type"`
	Version   int    `yaml:"version"`
	Timestamp string `yaml:"timestamp"`
	Payload   any    `yaml:"payload"`
}

type userCreatedPayloadOutput struct {
	ID        int    `yaml:"id"`
	Email     string `yaml:"email"`
	CreatedAt string `yaml:"created_at"`
}

type userUpdatedPayloadOutput struct {
	ID    int    `yaml:"id"`
	Email string `yaml:"email"`
}

type userDeletedPayloadOutput struct {
	ID int `yaml:"id"`
}

type messageCreatedPayloadOutput struct {
	MessageID int    `yaml:"message_id"`
	UserID    int    `yaml:"user_id"`
	Subject   string `yaml:"subject"`
	Content   string `yaml:"content"`
	DateSent  string `yaml:"date_sent"`
}

type messageDeletedPayloadOutput struct {
	MessageID int `yaml:"message_id"`
	UserID    int `yaml:"user_id"`
}

type contactAddedPayloadOutput struct {
	ContactID  int    `yaml:"contact_id"`
	UserID     int    `yaml:"user_id"`
	Value      string `yaml:"value"`
	Category   string `yaml:"category"`
	Importance int    `yaml:"importance"`
	CreatedAt  string `yaml:"created_at"`
}

type contactUpdatedPayloadOutput struct {
	ContactID     int    `yaml:"contact_id"`
	Value         string `yaml:"value"`
	OldValue      string `yaml:"old_value"`
	UserID        int    `yaml:"user_id"`
	OldCategory   string `yaml:"old_category"`
	NewCategory   string `yaml:"new_category"`
	OldImportance int    `yaml:"old_importance"`
	NewImportance int    `yaml:"new_importance"`
	UpdatedAt     string `yaml:"updated_at"`
}

type contactRemovedPayloadOutput struct {
	ContactID int    `yaml:"contact_id"`
	Value     string `yaml:"value"`
	UserID    int    `yaml:"user_id"`
}

type userResponseOutput struct {
	ID        int64  `json:"id" yaml:"id"`
	Email     string `json:"email" yaml:"email"`
	CreatedAt string `json:"created_at" yaml:"created_at"`
}

type userViewResponseOutput struct {
	ID                int64                            `json:"id" yaml:"id"`
	Email             string                           `json:"email" yaml:"email"`
	NumMessages       int                              `json:"num_messages" yaml:"num_messages"`
	ImportantContacts []readmodel.ImportantContactView `json:"important_contacts" yaml:"important_contacts"`
	Messages          []messageViewOutput              `json:"messages" yaml:"messages"`
	CreatedAt         string                           `json:"created_at" yaml:"created_at"`
}

type messageViewOutput struct {
	ID        int64  `json:"id" yaml:"id"`
	Subject   string `json:"subject" yaml:"subject"`
	Text      string `json:"text" yaml:"text"`
	CreatedAt string `json:"created_at" yaml:"created_at"`
}

type messageResponseOutput struct {
	ID         int64  `json:"id" yaml:"id"`
	ExternalID string `json:"external_id" yaml:"external_id"`
	SenderID   int64  `json:"sender_id" yaml:"sender_id"`
	ReceiverID int64  `json:"receiver_id" yaml:"receiver_id"`
	Subject    string `json:"subject" yaml:"subject"`
	Text       string `json:"text" yaml:"text"`
	DateSent   string `json:"date_sent" yaml:"date_sent"`
	CreatedAt  string `json:"created_at" yaml:"created_at"`
}

type contactOperationResponseOutput struct {
	Contact     contactDTO     `yaml:"contact"`
	UserContact userContactDTO `yaml:"user_contact"`
}

type contactDTO struct {
	ID        int64  `yaml:"ID"`
	Value     string `yaml:"Value"`
	CreatedAt string `yaml:"CreatedAt"`
}

type userContactDTO struct {
	ID         int64  `yaml:"ID"`
	UserID     int64  `yaml:"UserID"`
	ContactID  int64  `yaml:"ContactID"`
	Importance int    `yaml:"Importance"`
	Category   int    `yaml:"Category"`
	CreatedAt  string `yaml:"CreatedAt"`
}

func stringValue(value any) string {
	s, _ := value.(string)
	return s
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case uint64:
		return int(v)
	case int64:
		return int(v)
	default:
		return 0
	}
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case uint64:
		return int64(v)
	default:
		return 0
	}
}

func orderedKafkaPayload(root map[string]any) any {
	payload, ok := root["payload"].(map[string]any)
	if !ok {
		return root["payload"]
	}

	switch stringValue(root["event_type"]) {
	case "user_created":
		return userCreatedPayloadOutput{
			ID:        intValue(payload["id"]),
			Email:     stringValue(payload["email"]),
			CreatedAt: stringValue(payload["created_at"]),
		}
	case "user_updated":
		return userUpdatedPayloadOutput{
			ID:    intValue(payload["id"]),
			Email: stringValue(payload["email"]),
		}
	case "user_deleted":
		return userDeletedPayloadOutput{
			ID: intValue(payload["id"]),
		}
	case "message_created":
		return messageCreatedPayloadOutput{
			MessageID: intValue(payload["message_id"]),
			UserID:    intValue(payload["user_id"]),
			Subject:   stringValue(payload["subject"]),
			Content:   stringValue(payload["content"]),
			DateSent:  normalizeRFC3339(stringValue(payload["date_sent"])),
		}
	case "message_deleted":
		return messageDeletedPayloadOutput{
			MessageID: intValue(payload["message_id"]),
			UserID:    intValue(payload["user_id"]),
		}
	case "contact_added":
		return contactAddedPayloadOutput{
			ContactID:  intValue(payload["contact_id"]),
			UserID:     intValue(payload["user_id"]),
			Value:      stringValue(payload["value"]),
			Category:   stringValue(payload["category"]),
			Importance: intValue(payload["importance"]),
			CreatedAt:  stringValue(payload["created_at"]),
		}
	case "contact_updated":
		return contactUpdatedPayloadOutput{
			ContactID:     intValue(payload["contact_id"]),
			Value:         stringValue(payload["value"]),
			OldValue:      stringValue(payload["old_value"]),
			UserID:        intValue(payload["user_id"]),
			OldCategory:   stringValue(payload["old_category"]),
			NewCategory:   stringValue(payload["new_category"]),
			OldImportance: intValue(payload["old_importance"]),
			NewImportance: intValue(payload["new_importance"]),
			UpdatedAt:     stringValue(payload["updated_at"]),
		}
	case "contact_removed":
		return contactRemovedPayloadOutput{
			ContactID: intValue(payload["contact_id"]),
			Value:     stringValue(payload["value"]),
			UserID:    intValue(payload["user_id"]),
		}
	default:
		return payload
	}
}

func firstMapValue(m map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			return value
		}
	}
	return nil
}

func normalizeRFC3339(value string) string {
	if value == "" {
		return value
	}
	tm, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return tm.UTC().Format(time.RFC3339)
}
