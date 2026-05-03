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

type addContactRequest struct {
	ContactID  int64  `yaml:"contact_id"`
	Value      string `yaml:"value"`
	Importance int    `yaml:"importance"`
	Category   int    `yaml:"category"`
}

func (h *Handler) registerContactRoutes() {
	h.router.POST("/users/:user_id/contacts", h.addContact)
	h.router.PUT("/users/:user_id/contacts/:contact_id", h.updateContact)
	h.router.DELETE("/users/:user_id/contacts/:contact_id", h.deleteContact)
}

func (h *Handler) addContact(w nethttp.ResponseWriter, req bunrouter.Request) error {
	userID, err := pathInt64(req, "user_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse user_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	var body addContactRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	contact := &bunmodel.Contact{ID: body.ContactID, Value: body.Value}
	userContact := &bunmodel.UserContact{UserID: userID, ContactID: body.ContactID, Importance: body.Importance, Category: body.Category}
	if err := h.contacts.AddUserContact(req.Context(), userContact, contact); err != nil {
		nethttp.Error(w, fmt.Sprintf("add contact: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	if err := h.producer.Publish(req.Context(), events.ContactAdded, events.ContactAddedPayload{
		ContactID:  contact.ID,
		UserID:     userID,
		Value:      contact.Value,
		Category:   strconv.Itoa(userContact.Category),
		Importance: userContact.Importance,
		CreatedAt:  userContact.CreatedAt,
	}); err != nil {
		nethttp.Error(w, fmt.Sprintf("publish contact_added: %v", err), nethttp.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(nethttp.StatusCreated)
	return bunrouter.JSON(w, struct {
		Contact     *bunmodel.Contact     `json:"contact"`
		UserContact *bunmodel.UserContact `json:"user_contact"`
	}{Contact: contact, UserContact: userContact})
}

func (h *Handler) updateContact(w nethttp.ResponseWriter, req bunrouter.Request) error {
	userID, err := pathInt64(req, "user_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse user_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}
	contactID, err := pathInt64(req, "contact_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse contact_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	var body addContactRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	contact := &bunmodel.Contact{ID: contactID, Value: body.Value}
	userContact := &bunmodel.UserContact{UserID: userID, ContactID: contactID, Importance: body.Importance, Category: body.Category}
	before, err := h.contacts.UpdateUserContact(req.Context(), userContact, contact)
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("update contact: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	if err := h.producer.Publish(req.Context(), events.ContactUpdated, events.ContactUpdatedPayload{
		ContactID:     contact.ID,
		Value:         contact.Value,
		OldValue:      before.OldValue,
		UserID:        userID,
		OldCategory:   strconv.Itoa(before.OldCategory),
		NewCategory:   strconv.Itoa(userContact.Category),
		OldImportance: before.OldImportance,
		NewImportance: userContact.Importance,
		UpdatedAt:     time.Now().UTC(),
	}); err != nil {
		nethttp.Error(w, fmt.Sprintf("publish contact_updated: %v", err), nethttp.StatusInternalServerError)
		return nil
	}

	return bunrouter.JSON(w, struct {
		Contact     *bunmodel.Contact     `json:"contact"`
		UserContact *bunmodel.UserContact `json:"user_contact"`
	}{Contact: contact, UserContact: userContact})
}

func (h *Handler) deleteContact(w nethttp.ResponseWriter, req bunrouter.Request) error {
	userID, err := pathInt64(req, "user_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse user_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}
	contactID, err := pathInt64(req, "contact_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse contact_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	deleted, err := h.contacts.DeleteUserContact(req.Context(), userID, contactID)
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("delete contact: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	if err := h.producer.Publish(req.Context(), events.ContactRemoved, events.ContactRemovedPayload{
		ContactID: deleted.Contact.ID,
		Value:     deleted.Contact.Value,
		UserID:    userID,
	}); err != nil {
		nethttp.Error(w, fmt.Sprintf("publish contact_removed: %v", err), nethttp.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(nethttp.StatusNoContent)
	return nil
}
