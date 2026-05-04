package http

import (
	"bytes"
	"context"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"backend/internal/bunmodel"
	"backend/internal/readmodel"
	backendstorage "backend/internal/storage"
)

// TestHandler_CreateUser verifies POST /users writes a user.
// Event publishing is now handled by PostgreSQL triggers, not the handler.
func TestHandler_CreateUser(t *testing.T) {
	t.Helper()

	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_CreateThenReadUserTwice verifies that two consecutive reads after write
// return the same user view from the read-model store.
func TestHandler_CreateThenReadUserTwice(t *testing.T) {
	var createdUser bunmodel.User

	handler := newTestHandler(t, &stubDeps{
		users: &stubUserStore{
			createUserFn: func(_ context.Context, user *bunmodel.User) error {
				user.ID = 101
				user.CreatedAt = time.Date(2026, 5, 3, 12, 30, 0, 0, time.UTC)
				createdUser = *user
				return nil
			},
		},
		userViews: &stubUserViewStore{
			getUserViewByIDFn: func(_ context.Context, id int64) (*readmodel.UserView, error) {
				if id != createdUser.ID {
					return nil, nethttp.ErrMissingFile
				}
				return &readmodel.UserView{
					ID:          createdUser.ID,
					Email:       createdUser.Email,
					CreatedAt:   createdUser.CreatedAt,
					NumMessages: 0,
				}, nil
			},
		},
	})

	createReq := httptest.NewRequest(nethttp.MethodPost, "/users", bytes.NewBufferString("email: twice@example.com\n"))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	if createRec.Code != nethttp.StatusCreated {
		t.Fatalf("create status = %d, want %d, body=%s", createRec.Code, nethttp.StatusCreated, createRec.Body.String())
	}

	readReq1 := httptest.NewRequest(nethttp.MethodGet, "/users/101", nil)
	readRec1 := httptest.NewRecorder()
	handler.ServeHTTP(readRec1, readReq1)
	if readRec1.Code != nethttp.StatusOK {
		t.Fatalf("first read status = %d, want %d, body=%s", readRec1.Code, nethttp.StatusOK, readRec1.Body.String())
	}

	readReq2 := httptest.NewRequest(nethttp.MethodGet, "/users/101", nil)
	readRec2 := httptest.NewRecorder()
	handler.ServeHTTP(readRec2, readReq2)
	if readRec2.Code != nethttp.StatusOK {
		t.Fatalf("second read status = %d, want %d, body=%s", readRec2.Code, nethttp.StatusOK, readRec2.Body.String())
	}

	if readRec1.Body.String() != readRec2.Body.String() {
		t.Fatalf("read bodies differ:\n1=%s\n2=%s", readRec1.Body.String(), readRec2.Body.String())
	}
}

// TestHandler_CORSPreflight verifies OPTIONS preflight returns CORS headers.
func TestHandler_CORSPreflight(t *testing.T) {
	handler := newTestHandler(t, nil)

	req := httptest.NewRequest(nethttp.MethodOptions, "/users", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", nethttp.MethodPost)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusNoContent)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow-origin = %q, want %q", got, "http://localhost:5173")
	}
}

// TestHandler_CORSHeadersOnRequest verifies CORS headers on allowed-origin API calls.
func TestHandler_CORSHeadersOnRequest(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
		userViews: &stubUserViewStore{
			getUserViewByIDFn: func(_ context.Context, id int64) (*readmodel.UserView, error) {
				return &readmodel.UserView{ID: id, Email: "cors@example.com"}, nil
			},
		},
	})

	req := httptest.NewRequest(nethttp.MethodGet, "/users/1", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusOK)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow-origin = %q, want %q", got, "http://localhost:5173")
	}
}

// TestHandler_GetUserReadsMongoView verifies GET /users/:id reads the CQRS read model.
func TestHandler_GetUserReadsMongoView(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_UpdateUser verifies PUT /users/:id updates user.
func TestHandler_UpdateUser(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_DeleteUser verifies DELETE /users/:id deletes user.
func TestHandler_DeleteUser(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_CreateMessage verifies POST /messages creates message.
func TestHandler_CreateMessage(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_UpdateMessage verifies PUT /messages/:id updates message.
func TestHandler_UpdateMessage(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_DeleteMessage verifies DELETE /messages/:id deletes message.
func TestHandler_DeleteMessage(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_AddContact verifies POST /users/:id/contacts adds contact.
func TestHandler_AddContact(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_UpdateContact verifies PUT /users/:uid/contacts/:id updates contact.
func TestHandler_UpdateContact(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

// TestHandler_DeleteContact verifies DELETE /users/:uid/contacts/:id deletes contact.
func TestHandler_DeleteContact(t *testing.T) {
	handler := newTestHandler(t, &stubDeps{
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
}

type stubDeps struct {
	users     *stubUserStore
	messages  *stubMessageStore
	contacts  *stubContactStore
	userViews *stubUserViewStore
}

func newTestHandler(t *testing.T, deps *stubDeps) *Handler {
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

type stubUserStore struct {
	createUserFn func(ctx context.Context, user *bunmodel.User) error
	updateUserFn func(ctx context.Context, user *bunmodel.User) error
	deleteUserFn func(ctx context.Context, id int64) error
	listUsersFn  func(ctx context.Context) ([]bunmodel.User, error)
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

// ListUsers returns users from the stub store.
func (s *stubUserStore) ListUsers(ctx context.Context) ([]bunmodel.User, error) {
	if s.listUsersFn == nil {
		return []bunmodel.User{}, nil
	}
	return s.listUsersFn(ctx)
}

type stubMessageStore struct {
	createMessageFn func(ctx context.Context, message *bunmodel.Message) error
	updateMessageFn func(ctx context.Context, message *bunmodel.Message) error
	deleteMessageFn func(ctx context.Context, id int64) (*bunmodel.Message, error)
	listMessagesFn  func(ctx context.Context) ([]bunmodel.Message, error)
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

// ListMessages returns messages from the stub store.
func (s *stubMessageStore) ListMessages(ctx context.Context) ([]bunmodel.Message, error) {
	if s.listMessagesFn == nil {
		return []bunmodel.Message{}, nil
	}
	return s.listMessagesFn(ctx)
}

type stubContactStore struct {
	addUserContactFn    func(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) error
	updateUserContactFn func(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) (*backendstorage.ContactUpdateSnapshot, error)
	deleteUserContactFn func(ctx context.Context, userID int64, contactID int64) (*backendstorage.DeletedUserContact, error)
	listUserContactsFn  func(ctx context.Context) ([]backendstorage.UserContactWithValue, error)
	listByUserIDFn      func(ctx context.Context, userID int64) ([]backendstorage.UserContactWithValue, error)
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

// ListUserContacts returns contact links from the stub store.
func (s *stubContactStore) ListUserContacts(ctx context.Context) ([]backendstorage.UserContactWithValue, error) {
	if s.listUserContactsFn == nil {
		return []backendstorage.UserContactWithValue{}, nil
	}
	return s.listUserContactsFn(ctx)
}

// ListUserContactsByUserID returns user-specific contact links from the stub store.
func (s *stubContactStore) ListUserContactsByUserID(ctx context.Context, userID int64) ([]backendstorage.UserContactWithValue, error) {
	if s.listByUserIDFn == nil {
		return []backendstorage.UserContactWithValue{}, nil
	}
	return s.listByUserIDFn(ctx, userID)
}

type stubUserViewStore struct {
	getUserViewByIDFn func(ctx context.Context, id int64) (*readmodel.UserView, error)
	listUserViewsFn   func(ctx context.Context) ([]readmodel.UserView, error)
	listMessagesFn    func(ctx context.Context) ([]readmodel.MessageRow, error)
	listContactsFn    func(ctx context.Context) ([]readmodel.ContactRow, error)
	listByUserIDFn    func(ctx context.Context, userID int64) ([]readmodel.ContactRow, error)
}

// GetUserViewByID returns a stub read model user.
func (s *stubUserViewStore) GetUserViewByID(ctx context.Context, id int64) (*readmodel.UserView, error) {
	if s.getUserViewByIDFn == nil {
		return &readmodel.UserView{}, nil
	}
	return s.getUserViewByIDFn(ctx, id)
}

// ListUserViews returns projected users from the stub store.
func (s *stubUserViewStore) ListUserViews(ctx context.Context) ([]readmodel.UserView, error) {
	if s.listUserViewsFn == nil {
		return []readmodel.UserView{}, nil
	}
	return s.listUserViewsFn(ctx)
}

// ListMessageViews returns projected messages from the stub store.
func (s *stubUserViewStore) ListMessageViews(ctx context.Context) ([]readmodel.MessageRow, error) {
	if s.listMessagesFn == nil {
		return []readmodel.MessageRow{}, nil
	}
	return s.listMessagesFn(ctx)
}

// ListContactViews returns projected contacts from the stub store.
func (s *stubUserViewStore) ListContactViews(ctx context.Context) ([]readmodel.ContactRow, error) {
	if s.listContactsFn == nil {
		return []readmodel.ContactRow{}, nil
	}
	return s.listContactsFn(ctx)
}

// ListContactViewsByUserID returns projected contacts for one user from the stub store.
func (s *stubUserViewStore) ListContactViewsByUserID(ctx context.Context, userID int64) ([]readmodel.ContactRow, error) {
	if s.listByUserIDFn == nil {
		return []readmodel.ContactRow{}, nil
	}
	return s.listByUserIDFn(ctx, userID)
}
