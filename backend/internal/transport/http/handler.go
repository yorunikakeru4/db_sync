package http

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"strconv"
	"strings"

	"backend/internal/bunmodel"
	"backend/internal/readmodel"
	backendstorage "backend/internal/storage"

	"github.com/uptrace/bunrouter"
)

// UserStore persists user mutations.
type UserStore interface {
	// CreateUser inserts a user and mutates it with database-generated fields.
	CreateUser(ctx context.Context, user *bunmodel.User) error
	// UpdateUser updates a user and mutates it with the persisted fields.
	UpdateUser(ctx context.Context, user *bunmodel.User) error
	// DeleteUser removes a user by primary key.
	DeleteUser(ctx context.Context, id int64) error
}

// UserViewStore reads CQRS user projections from MongoDB.
type UserViewStore interface {
	// GetUserViewByID returns the denormalized MongoDB user document.
	GetUserViewByID(ctx context.Context, id int64) (*readmodel.UserView, error)
	// ListUserViews returns all denormalized MongoDB user documents.
	ListUserViews(ctx context.Context) ([]readmodel.UserView, error)
	// ListMessageViews returns flattened message rows from MongoDB projections.
	ListMessageViews(ctx context.Context) ([]readmodel.MessageRow, error)
	// ListContactViews returns flattened contact rows from MongoDB projections.
	ListContactViews(ctx context.Context) ([]readmodel.ContactRow, error)
	// ListContactViewsByUserID returns flattened contact rows for a user projection.
	ListContactViewsByUserID(ctx context.Context, userID int64) ([]readmodel.ContactRow, error)
}

// MessageStore persists message mutations.
type MessageStore interface {
	// CreateMessage inserts a message and mutates it with database-generated fields.
	CreateMessage(ctx context.Context, message *bunmodel.Message) error
	// UpdateMessage updates a message and mutates it with the persisted fields.
	UpdateMessage(ctx context.Context, message *bunmodel.Message) error
	// DeleteMessage removes a message by primary key and returns the deleted row.
	DeleteMessage(ctx context.Context, id int64) (*bunmodel.Message, error)
}

// ContactStore persists contact and user-contact mutations.
type ContactStore interface {
	// AddUserContact creates or reuses a contact and links it to a user.
	AddUserContact(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) error
	// UpdateUserContact updates both the contact row and association metadata.
	UpdateUserContact(ctx context.Context, userContact *bunmodel.UserContact, contact *bunmodel.Contact) (*backendstorage.ContactUpdateSnapshot, error)
	// DeleteUserContact removes the user-contact link and returns the deleted state.
	DeleteUserContact(ctx context.Context, userID, contactID int64) (*backendstorage.DeletedUserContact, error)
}

// HandlerParams contains handler dependencies.
type HandlerParams struct {
	// Users persists user mutations.
	Users UserStore
	// Messages persists message mutations.
	Messages MessageStore
	// Contacts persists contact mutations.
	Contacts ContactStore
	// UserViews reads CQRS user projections.
	UserViews UserViewStore
}

// Handler serves backend write-side HTTP requests.
type Handler struct {
	users     UserStore
	messages  MessageStore
	contacts  ContactStore
	userViews UserViewStore
	router    *bunrouter.Router
}

// NewHandler builds the backend HTTP router.
func NewHandler(params HandlerParams) (*Handler, error) {
	if params.Users == nil {
		return nil, errors.New("user store is required")
	}
	if params.Messages == nil {
		return nil, errors.New("message store is required")
	}
	if params.Contacts == nil {
		return nil, errors.New("contact store is required")
	}
	if params.UserViews == nil {
		return nil, errors.New("user view store is required")
	}

	h := &Handler{
		users:     params.Users,
		messages:  params.Messages,
		contacts:  params.Contacts,
		userViews: params.UserViews,
		router:    bunrouter.New(),
	}
	h.registerUserRoutes()
	h.registerMessageRoutes()
	h.registerContactRoutes()
	return h, nil
}

// ServeHTTP routes a backend HTTP request.
func (h *Handler) ServeHTTP(w nethttp.ResponseWriter, req *nethttp.Request) {
	if h.applyCORS(w, req) {
		return
	}
	h.router.ServeHTTP(w, req)
}

func (h *Handler) applyCORS(w nethttp.ResponseWriter, req *nethttp.Request) bool {
	origin := req.Header.Get("Origin")
	if !isAllowedOrigin(origin) {
		return false
	}

	headers := w.Header()
	headers.Set("Access-Control-Allow-Origin", origin)
	headers.Set("Vary", "Origin")
	headers.Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
	headers.Set("Access-Control-Allow-Headers", "Content-Type,Authorization")

	if req.Method == nethttp.MethodOptions {
		w.WriteHeader(nethttp.StatusNoContent)
		return true
	}
	return false
}

func isAllowedOrigin(origin string) bool {
	switch strings.TrimSpace(origin) {
	case "http://localhost:5173", "http://127.0.0.1:5173":
		return true
	default:
		return false
	}
}

func pathInt64(req bunrouter.Request, key string) (int64, error) {
	value, err := strconv.ParseInt(req.Param(key), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}
