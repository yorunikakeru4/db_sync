package http

import (
	"context"
	"errors"
	"fmt"
	nethttp "net/http"
	"strconv"

	"backend/internal/bunmodel"
	"backend/internal/kafka"
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

// EmailStore persists email and user-email mutations.
type EmailStore interface {
	// AddUserEmail creates or reuses an email and links it to a user.
	AddUserEmail(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) error
	// UpdateUserEmail updates both the email row and association metadata.
	UpdateUserEmail(ctx context.Context, userEmail *bunmodel.UserEmail, email *bunmodel.Email) (*backendstorage.EmailUpdateSnapshot, error)
	// DeleteUserEmail removes the user-email link and returns the deleted state.
	DeleteUserEmail(ctx context.Context, userID, emailID int64) (*backendstorage.DeletedUserEmail, error)
}

// HandlerParams contains handler dependencies.
type HandlerParams struct {
	// Producer publishes mutation events.
	Producer kafka.Producer
	// Users persists user mutations.
	Users UserStore
	// Messages persists message mutations.
	Messages MessageStore
	// Emails persists email mutations.
	Emails EmailStore
	// UserViews reads CQRS user projections.
	UserViews UserViewStore
}

// Handler serves backend write-side HTTP requests.
type Handler struct {
	producer kafka.Producer
	users    UserStore
	messages MessageStore
	emails   EmailStore
	userViews UserViewStore
	router   *bunrouter.Router
}

// NewHandler builds the backend HTTP router.
func NewHandler(params HandlerParams) (*Handler, error) {
	if params.Producer == nil {
		return nil, errors.New("producer is required")
	}
	if params.Users == nil {
		return nil, errors.New("user store is required")
	}
	if params.Messages == nil {
		return nil, errors.New("message store is required")
	}
	if params.Emails == nil {
		return nil, errors.New("email store is required")
	}
	if params.UserViews == nil {
		return nil, errors.New("user view store is required")
	}

	h := &Handler{
		producer: params.Producer,
		users:    params.Users,
		messages: params.Messages,
		emails:   params.Emails,
		userViews: params.UserViews,
		router:   bunrouter.New(),
	}
	h.registerUserRoutes()
	h.registerMessageRoutes()
	h.registerEmailRoutes()
	return h, nil
}

// ServeHTTP routes a backend HTTP request.
func (h *Handler) ServeHTTP(w nethttp.ResponseWriter, req *nethttp.Request) {
	h.router.ServeHTTP(w, req)
}

func pathInt64(req bunrouter.Request, key string) (int64, error) {
	value, err := strconv.ParseInt(req.Param(key), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return value, nil
}
