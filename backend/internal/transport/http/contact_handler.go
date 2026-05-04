package http

import (
	"fmt"
	nethttp "net/http"

	"backend/internal/bunmodel"

	"github.com/goccy/go-yaml"
	"github.com/uptrace/bunrouter"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type addContactRequest struct {
	ContactID  int64  `yaml:"contact_id"`
	Value      string `yaml:"value"`
	Importance int    `yaml:"importance"`
	Category   int    `yaml:"category"`
}

func (h *Handler) registerContactRoutes() {
	h.router.GET("/contacts", h.listContacts)
	h.router.GET("/users/:user_id/contacts", h.listContactsByUserID)
	h.router.POST("/users/:user_id/contacts", h.addContact)
	h.router.PUT("/users/:user_id/contacts/:contact_id", h.updateContact)
	h.router.DELETE("/users/:user_id/contacts/:contact_id", h.deleteContact)
}

func (h *Handler) listContacts(w nethttp.ResponseWriter, req bunrouter.Request) error {
	contacts, err := h.userViews.ListContactViews(req.Context())
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("list contacts: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	return bunrouter.JSON(w, contacts)
}

func (h *Handler) listContactsByUserID(w nethttp.ResponseWriter, req bunrouter.Request) error {
	userID, err := pathInt64(req, "user_id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse user_id: %v", err), nethttp.StatusBadRequest)
		return nil
	}
	contacts, err := h.userViews.ListContactViewsByUserID(req.Context(), userID)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			nethttp.Error(w, "user not found", nethttp.StatusNotFound)
			return nil
		}
		nethttp.Error(w, fmt.Sprintf("list contacts by user: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	return bunrouter.JSON(w, contacts)
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
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

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
	_, err = h.contacts.UpdateUserContact(req.Context(), userContact, contact)
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("update contact: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

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

	_, err = h.contacts.DeleteUserContact(req.Context(), userID, contactID)
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("delete contact: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

	w.WriteHeader(nethttp.StatusNoContent)
	return nil
}
