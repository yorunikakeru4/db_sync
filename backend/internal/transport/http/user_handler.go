package http

import (
	"errors"
	"fmt"
	nethttp "net/http"
	"time"

	"backend/internal/bunmodel"
	backendstorage "backend/internal/storage"

	"github.com/goccy/go-yaml"
	"github.com/uptrace/bunrouter"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type createUserRequest struct {
	// Email is the user's email address.
	Email string `yaml:"email"`
}

func (h *Handler) registerUserRoutes() {
	h.router.GET("/users", h.listUsers)
	h.router.GET("/users/:id", h.getUser)
	h.router.POST("/users", h.createUser)
	h.router.PUT("/users/:id", h.updateUser)
	h.router.DELETE("/users/:id", h.deleteUser)
}

func (h *Handler) listUsers(w nethttp.ResponseWriter, req bunrouter.Request) error {
	users, err := h.userViews.ListUserViews(req.Context())
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("list users: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	return bunrouter.JSON(w, users)
}

func (h *Handler) getUser(w nethttp.ResponseWriter, req bunrouter.Request) error {
	id, err := pathInt64(req, "id")
	if err != nil {
		nethttp.Error(w, err.Error(), nethttp.StatusBadRequest)
		return nil
	}

	user, err := h.userViews.GetUserViewByID(req.Context(), id)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			nethttp.Error(w, "user not found", nethttp.StatusNotFound)
			return nil
		}
		nethttp.Error(w, fmt.Sprintf("get user view: %v", err), nethttp.StatusInternalServerError)
		return nil
	}

	return bunrouter.JSON(w, user)
}

func (h *Handler) createUser(w nethttp.ResponseWriter, req bunrouter.Request) error {
	var body createUserRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}
	if body.Email == "" {
		nethttp.Error(w, "email is required", nethttp.StatusBadRequest)
		return nil
	}

	user := &bunmodel.User{Email: body.Email, CreatedAt: time.Now().UTC()}
	if err := h.users.CreateUser(req.Context(), user); err != nil {
		nethttp.Error(w, fmt.Sprintf("create user: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

	w.WriteHeader(nethttp.StatusCreated)
	return bunrouter.JSON(w, user)
}

func (h *Handler) updateUser(w nethttp.ResponseWriter, req bunrouter.Request) error {
	id, err := pathInt64(req, "id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	var body createUserRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	user := &bunmodel.User{ID: id, Email: body.Email}
	if err := h.users.UpdateUser(req.Context(), user); err != nil {
		nethttp.Error(w, fmt.Sprintf("update user: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

	return bunrouter.JSON(w, user)
}

func (h *Handler) deleteUser(w nethttp.ResponseWriter, req bunrouter.Request) error {
	id, err := pathInt64(req, "id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	if err := h.users.DeleteUser(req.Context(), id); err != nil {
		if errors.Is(err, backendstorage.ErrUserNotFound) {
			nethttp.Error(w, "user not found", nethttp.StatusNotFound)
			return nil
		}
		nethttp.Error(w, fmt.Sprintf("delete user: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

	w.WriteHeader(nethttp.StatusNoContent)
	return nil
}
