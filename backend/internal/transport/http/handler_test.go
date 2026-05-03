package http

import (
	"bytes"
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/bunmodel"
	"backend/internal/events"
	"backend/internal/kafka"
	"backend/internal/readmodel"
	backendstorage "backend/internal/storage"
)

// TestHandler_CreateUserPublishesEvent verifies POST /users writes a user and emits user_created.
func TestHandler_CreateUserPublishesEvent(t *testing.T) {
	t.Helper()

	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		users: &stubUserStore{
			createUserFn: func(_ context.Context, user *bunmodel.User) error {
				user.ID = 42
				user.CreatedAt = time.Date(2026, 5, 3, 10, 0, 0, 0, time.UTC)
				return nil
			},
		},
	})

	req := httptest.NewRequestWithContext(
		context.Background(),
		nethttp.MethodPost,
		"/users",
		bytes.NewBufferString("email: user@example.com\n"),
	)
	req.Header.Set("Content-Type", "application/x-yaml")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusCreated, rec.Body.String())
	}
	if len(producer.Events) != 1 {
		t.Fatalf("published = %d, want 1", len(producer.Events))
	}
	if producer.Events[0].EventType != events.UserCreated {
		t.Fatalf("event_type = %q, want %q", producer.Events[0].EventType, events.UserCreated)
	}
}

// TestHandler_GetUserReadsMongoView verifies GET /users/:id reads the CQRS read model.
func TestHandler_GetUserReadsMongoView(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		userViews: &stubUserViewStore{
			getUserViewByIDFn: func(_ context.Context, id int64) (*readmodel.UserView, error) {
				return &readmodel.UserView{ID: id, Email: "read@example.com", NumMessages: 3}, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/users/7", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusOK, rec.Body.String())
	}
	if producer.Events != nil && len(producer.Events) != 0 {
		t.Fatalf("published read events = %d, want 0", len(producer.Events))
	}
}

// TestHandler_UpdateUserPublishesEvent verifies PUT /users/:id emits user_updated.
func TestHandler_UpdateUserPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		users: &stubUserStore{
			updateUserFn: func(_ context.Context, user *bunmodel.User) error {
				return nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodPut, "/users/9", bytes.NewBufferString("email: next@example.com\n"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusOK, rec.Body.String())
	}
	assertEventType(t, producer, events.UserUpdated)
}

// TestHandler_DeleteUserPublishesEvent verifies DELETE /users/:id emits user_deleted.
func TestHandler_DeleteUserPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		users: &stubUserStore{
			deleteUserFn: func(_ context.Context, id int64) error { return nil },
		},
	})

	req := httptest.NewRequest(nethttp.MethodDelete, "/users/9", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusNoContent, rec.Body.String())
	}
	assertEventType(t, producer, events.UserDeleted)
}

// TestHandler_CreateMessagePublishesEvent verifies POST /messages emits message_created.
func TestHandler_CreateMessagePublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		messages: &stubMessageStore{
			createMessageFn: func(_ context.Context, message *bunmodel.Message) error {
				message.ID = 77
				return nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodPost, "/messages", bytes.NewBufferString(
		"external_id: ext-1\nsender_id: 1\nreceiver_id: 2\nsubject: hi\ntext: body\ndate_sent: 2026-05-03T10:00:00Z\n",
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusCreated, rec.Body.String())
	}
	assertEventType(t, producer, events.MessageCreated)
}

// TestHandler_UpdateMessagePublishesEvent verifies PUT /messages/:id emits message_created.
func TestHandler_UpdateMessagePublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		messages: &stubMessageStore{
			updateMessageFn: func(_ context.Context, message *bunmodel.Message) error { return nil },
		},
	})

	req := httptest.NewRequest(nethttp.MethodPut, "/messages/77", bytes.NewBufferString(
		"external_id: ext-2\nsender_id: 3\nreceiver_id: 4\nsubject: upd\ntext: text\ndate_sent: 2026-05-03T11:00:00Z\n",
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusOK, rec.Body.String())
	}
	assertEventType(t, producer, events.MessageCreated)
}

// TestHandler_DeleteMessagePublishesEvent verifies DELETE /messages/:id emits message_deleted.
func TestHandler_DeleteMessagePublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		messages: &stubMessageStore{
			deleteMessageFn: func(_ context.Context, id int64) (*bunmodel.Message, error) {
				return &bunmodel.Message{ID: id, SenderID: 5}, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodDelete, "/messages/77", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusNoContent, rec.Body.String())
	}
	assertEventType(t, producer, events.MessageDeleted)
}

// TestHandler_AddContactPublishesEvent verifies POST /users/:id/contacts emits contact_added.
func TestHandler_AddContactPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		contacts: &stubContactStore{
			addUserContactFn: func(_ context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) error {
				contact.ID = 8
				userContact.CreatedAt = time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
				return nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodPost, "/users/1/contacts", bytes.NewBufferString(
		"contact_id: 8\nvalue: a@example.com\nimportance: 3\ncategory: 7\n",
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusCreated, rec.Body.String())
	}
	assertEventType(t, producer, events.ContactAdded)
}

// TestHandler_UpdateContactPublishesEvent verifies PUT /users/:uid/contacts/:id emits contact_updated.
func TestHandler_UpdateContactPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		contacts: &stubContactStore{
			updateUserContactFn: func(_ context.Context, _ *bunmodel.UserContact, _ *bunmodel.Contact) (*backendstorage.ContactUpdateSnapshot, error) {
				return &backendstorage.ContactUpdateSnapshot{OldValue: "before@example.com", OldCategory: 1, OldImportance: 2}, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodPut, "/users/1/contacts/8", bytes.NewBufferString(
		"value: a@example.com\nimportance: 5\ncategory: 9\n",
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusOK, rec.Body.String())
	}
	assertEventType(t, producer, events.ContactUpdated)
}

// TestHandler_DeleteContactPublishesEvent verifies DELETE /users/:uid/contacts/:id emits contact_removed.
func TestHandler_DeleteContactPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		contacts: &stubContactStore{
			deleteUserContactFn: func(_ context.Context, _ int64, id int64) (*backendstorage.DeletedUserContact, error) {
				return &backendstorage.DeletedUserContact{Contact: bunmodel.Contact{ID: id, Value: "gone@example.com"}}, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodDelete, "/users/1/contacts/8", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusNoContent, rec.Body.String())
	}
	assertEventType(t, producer, events.ContactRemoved)
}

type stubDeps struct {
	users     *stubUserStore
	messages  *stubMessageStore
	contacts  *stubContactStore
	userViews *stubUserViewStore
}

func newTestHandler(t *testing.T, producer *kafka.MemProducer, deps *stubDeps) *Handler {
	t.Helper()

	if deps == nil {
		deps = &stubDeps{}
	}
	if deps.users == nil {
		deps.users = &stubUserStore{}
	}
	if deps.messages == nil {
		deps.messages = &stubMessageStore{}
	}
	if deps.contacts == nil {
		deps.contacts = &stubContactStore{}
	}
	if deps.userViews == nil {
		deps.userViews = &stubUserViewStore{}
	}

	handler, err := NewHandler(HandlerParams{
		Producer:  producer,
		Users:     deps.users,
		Messages:  deps.messages,
		Contacts:  deps.contacts,
		UserViews: deps.userViews,
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return handler
}

func assertEventType(t *testing.T, producer *kafka.MemProducer, want string) {
	t.Helper()
	if len(producer.Events) != 1 {
		t.Fatalf("published = %d, want 1", len(producer.Events))
	}
	if producer.Events[0].EventType != want {
		t.Fatalf("event_type = %q, want %q", producer.Events[0].EventType, want)
	}
}

type stubUserStore struct {
	createUserFn func(ctx context.Context, user *bunmodel.User) error
	updateUserFn func(ctx context.Context, user *bunmodel.User) error
	deleteUserFn func(ctx context.Context, id int64) error
}

// CreateUser inserts a user in the stub store.
func (s *stubUserStore) CreateUser(ctx context.Context, user *bunmodel.User) error {
	if s.createUserFn == nil {
		return nil
	}
	return s.createUserFn(ctx, user)
}

// UpdateUser updates a user in the stub store.
func (s *stubUserStore) UpdateUser(ctx context.Context, user *bunmodel.User) error {
	if s.updateUserFn == nil {
		return nil
	}
	return s.updateUserFn(ctx, user)
}

// DeleteUser deletes a user in the stub store.
func (s *stubUserStore) DeleteUser(ctx context.Context, id int64) error {
	if s.deleteUserFn == nil {
		return nil
	}
	return s.deleteUserFn(ctx, id)
}

type stubMessageStore struct {
	createMessageFn func(ctx context.Context, message *bunmodel.Message) error
	updateMessageFn func(ctx context.Context, message *bunmodel.Message) error
	deleteMessageFn func(ctx context.Context, id int64) (*bunmodel.Message, error)
}

// CreateMessage inserts a message in the stub store.
func (s *stubMessageStore) CreateMessage(ctx context.Context, message *bunmodel.Message) error {
	if s.createMessageFn == nil {
		return nil
	}
	return s.createMessageFn(ctx, message)
}

// UpdateMessage updates a message in the stub store.
func (s *stubMessageStore) UpdateMessage(ctx context.Context, message *bunmodel.Message) error {
	if s.updateMessageFn == nil {
		return nil
	}
	return s.updateMessageFn(ctx, message)
}

// DeleteMessage deletes a message in the stub store.
func (s *stubMessageStore) DeleteMessage(ctx context.Context, id int64) (*bunmodel.Message, error) {
	if s.deleteMessageFn == nil {
		return &bunmodel.Message{}, nil
	}
	return s.deleteMessageFn(ctx, id)
}

type stubContactStore struct {
	addUserContactFn    func(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) error
	updateUserContactFn func(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) (*backendstorage.ContactUpdateSnapshot, error)
	deleteUserContactFn func(ctx context.Context, userID int64, contactID int64) (*backendstorage.DeletedUserContact, error)
}

// AddUserContact inserts a user-contact link in the stub store.
func (s *stubContactStore) AddUserContact(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) error {
	if s.addUserContactFn == nil {
		return nil
	}
	return s.addUserContactFn(ctx, userContact, contact)
}

// UpdateUserContact updates a user-contact link in the stub store.
func (s *stubContactStore) UpdateUserContact(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) (*backendstorage.ContactUpdateSnapshot, error) {
	if s.updateUserContactFn == nil {
		return &backendstorage.ContactUpdateSnapshot{}, nil
	}
	return s.updateUserContactFn(ctx, userContact, contact)
}

// DeleteUserContact deletes a user-contact link in the stub store.
func (s *stubContactStore) DeleteUserContact(ctx context.Context, userID int64, contactID int64) (*backendstorage.DeletedUserContact, error) {
	if s.deleteUserContactFn == nil {
		return &backendstorage.DeletedUserContact{}, nil
	}
	return s.deleteUserContactFn(ctx, userID, contactID)
}

type stubUserViewStore struct {
	getUserViewByIDFn func(ctx context.Context, id int64) (*readmodel.UserView, error)
}

// GetUserViewByID returns a stub read model user.
func (s *stubUserViewStore) GetUserViewByID(ctx context.Context, id int64) (*readmodel.UserView, error) {
	if s.getUserViewByIDFn == nil {
		return &readmodel.UserView{}, nil
	}
	return s.getUserViewByIDFn(ctx, id)
}
