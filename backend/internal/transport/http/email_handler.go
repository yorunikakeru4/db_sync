package http

import (
	"fmt"
	nethttp "net/http"
	"strconv"
	"time"

	"backend/internal/bunmodel"
	"backend/internal/events"

	"github.com/goccy/go-yaml"
	"github.com/uptrace/bunrouter"
)

type addEmailRequest struct {
	EmailID    int64  `yaml:"email_id"`
	Address    string `yaml:"address"`
	Importance int    `yaml:"importance"`
	Category   int    `yaml:"category"`
}

func (h *Handler) registerEmailRoutes() {
	h.router.POST("/users/:user_id/emails", h.addEmail)
	h.router.PUT("/users/:user_id/emails/:email_id", h.updateEmail)
	h.router.DELETE("/users/:user_id/emails/:email_id", h.deleteEmail)
}

func (h *Handler) addEmail(w nethttp.ResponseWriter, req bunrouter.Request) error {
	userID, err := pathInt64(req, "user_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse user_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	var body addEmailRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	email := &bunmodel.Email{ID: body.EmailID, Address: body.Address}
	userEmail := &bunmodel.UserEmail{UserID: userID, EmailID: body.EmailID, Importance: body.Importance, Category: body.Category}
	if err := h.emails.AddUserEmail(req.Context(), userEmail, email); err != nil {
		nethttp.Error(w, fmt.Sprintf("add email: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	if err := h.producer.Publish(req.Context(), events.EmailAdded, events.EmailAddedPayload{
		EmailID:     email.ID,
		UserID:      userID,
		Address:     email.Address,
		Category:    strconv.Itoa(userEmail.Category),
		Importance:  userEmail.Importance,
		CreatedAt:   userEmail.CreatedAt,
	}); err != nil {
		nethttp.Error(w, fmt.Sprintf("publish email_added: %v", err), nethttp.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(nethttp.StatusCreated)
	return bunrouter.JSON(w, struct {
		Email     *bunmodel.Email     `json:"email"`
		UserEmail *bunmodel.UserEmail `json:"user_email"`
	}{Email: email, UserEmail: userEmail})
}

func (h *Handler) updateEmail(w nethttp.ResponseWriter, req bunrouter.Request) error {
	userID, err := pathInt64(req, "user_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse user_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}
	emailID, err := pathInt64(req, "email_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse email_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	var body addEmailRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	email := &bunmodel.Email{ID: emailID, Address: body.Address}
	userEmail := &bunmodel.UserEmail{UserID: userID, EmailID: emailID, Importance: body.Importance, Category: body.Category}
	before, err := h.emails.UpdateUserEmail(req.Context(), userEmail, email)
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("update email: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	if err := h.producer.Publish(req.Context(), events.EmailUpdated, events.EmailUpdatedPayload{
		EmailID:       email.ID,
		Address:       email.Address,
		OldAddress:    before.OldAddress,
		UserID:        userID,
		OldCategory:   strconv.Itoa(before.OldCategory),
		NewCategory:   strconv.Itoa(userEmail.Category),
		OldImportance: before.OldImportance,
		NewImportance: userEmail.Importance,
		UpdatedAt:     time.Now().UTC(),
	}); err != nil {
		nethttp.Error(w, fmt.Sprintf("publish email_updated: %v", err), nethttp.StatusInternalServerError)
		return nil
	}

	return bunrouter.JSON(w, struct {
		Email     *bunmodel.Email     `json:"email"`
		UserEmail *bunmodel.UserEmail `json:"user_email"`
	}{Email: email, UserEmail: userEmail})
}

func (h *Handler) deleteEmail(w nethttp.ResponseWriter, req bunrouter.Request) error {
	userID, err := pathInt64(req, "user_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse user_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}
	emailID, err := pathInt64(req, "email_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse email_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	deleted, err := h.emails.DeleteUserEmail(req.Context(), userID, emailID)
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("delete email: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	if err := h.producer.Publish(req.Context(), events.EmailRemoved, events.EmailRemovedPayload{
		EmailID: deleted.Email.ID,
		Address: deleted.Email.Address,
		UserID:  userID,
	}); err != nil {
		nethttp.Error(w, fmt.Sprintf("publish email_removed: %v", err), nethttp.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(nethttp.StatusNoContent)
	return nil
}
