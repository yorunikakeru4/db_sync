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

// TestHandler_AddEmailPublishesEvent verifies POST /users/:id/emails emits email_added.
func TestHandler_AddEmailPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		emails: &stubEmailStore{
			addUserEmailFn: func(_ context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) error {
				email.ID = 8
				userEmail.CreatedAt = time.Date(2026, 5, 3, 12, 0, 0, 0, time.UTC)
				return nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodPost, "/users/1/emails", bytes.NewBufferString(
		"email_id: 8\naddress: a@example.com\nimportance: 3\ncategory: 7\n",
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusCreated {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusCreated, rec.Body.String())
	}
	assertEventType(t, producer, events.EmailAdded)
}

// TestHandler_UpdateEmailPublishesEvent verifies PUT /users/:uid/emails/:id emits email_updated.
func TestHandler_UpdateEmailPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		emails: &stubEmailStore{
			updateUserEmailFn: func(_ context.Context, _ *bunmodel.UserEmail, _ *bunmodel.Email) (*backendstorage.EmailUpdateSnapshot, error) {
				return &backendstorage.EmailUpdateSnapshot{OldAddress: "before@example.com", OldCategory: 1, OldImportance: 2}, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodPut, "/users/1/emails/8", bytes.NewBufferString(
		"address: a@example.com\nimportance: 5\ncategory: 9\n",
	))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusOK, rec.Body.String())
	}
	assertEventType(t, producer, events.EmailUpdated)
}

// TestHandler_DeleteEmailPublishesEvent verifies DELETE /users/:uid/emails/:id emits email_removed.
func TestHandler_DeleteEmailPublishesEvent(t *testing.T) {
	producer := &kafka.MemProducer{}
	handler := newTestHandler(t, producer, &stubDeps{
		emails: &stubEmailStore{
			deleteUserEmailFn: func(_ context.Context, _ int64, id int64) (*backendstorage.DeletedUserEmail, error) {
				return &backendstorage.DeletedUserEmail{Email: bunmodel.Email{ID: id, Address: "gone@example.com"}}, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodDelete, "/users/1/emails/8", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("status = %d, want %d, body=%s", rec.Code, nethttp.StatusNoContent, rec.Body.String())
	}
	assertEventType(t, producer, events.EmailRemoved)
}

type stubDeps struct {
	users     *stubUserStore
	messages  *stubMessageStore
	emails    *stubEmailStore
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
	if deps.emails == nil {
		deps.emails = &stubEmailStore{}
	}
	if deps.userViews == nil {
		deps.userViews = &stubUserViewStore{}
	}

	handler, err := NewHandler(HandlerParams{
		Producer:  producer,
		Users:     deps.users,
		Messages:  deps.messages,
		Emails:    deps.emails,
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

type stubEmailStore struct {
	addUserEmailFn    func(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) error
	updateUserEmailFn func(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) (*backendstorage.EmailUpdateSnapshot, error)
	deleteUserEmailFn func(ctx context.Context, userID int64, emailID int64) (*backendstorage.DeletedUserEmail, error)
}

// AddUserEmail inserts a user-email link in the stub store.
func (s *stubEmailStore) AddUserEmail(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) error {
	if s.addUserEmailFn == nil {
		return nil
	}
	return s.addUserEmailFn(ctx, userEmail, email)
}

// UpdateUserEmail updates a user-email link in the stub store.
func (s *stubEmailStore) UpdateUserEmail(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) (*backendstorage.EmailUpdateSnapshot, error) {
	if s.updateUserEmailFn == nil {
		return &backendstorage.EmailUpdateSnapshot{}, nil
	}
	return s.updateUserEmailFn(ctx, userEmail, email)
}

// DeleteUserEmail deletes a user-email link in the stub store.
func (s *stubEmailStore) DeleteUserEmail(ctx context.Context, userID int64, emailID int64) (*backendstorage.DeletedUserEmail, error) {
	if s.deleteUserEmailFn == nil {
		return &backendstorage.DeletedUserEmail{}, nil
	}
	return s.deleteUserEmailFn(ctx, userID, emailID)
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
