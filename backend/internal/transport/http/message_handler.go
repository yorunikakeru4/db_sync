package http

import (
	"fmt"
	nethttp "net/http"
	"time"

	"backend/internal/bunmodel"

	"github.com/goccy/go-yaml"
	"github.com/uptrace/bunrouter"
)

type createMessageRequest struct {
	ExternalID string    `yaml:"external_id"`
	SenderID   int64     `yaml:"sender_id"`
	ReceiverID int64     `yaml:"receiver_id"`
	Subject    string    `yaml:"subject"`
	Text       string    `yaml:"text"`
	DateSent   time.Time `yaml:"date_sent"`
}

func (h *Handler) registerMessageRoutes() {
	h.router.GET("/messages", h.listMessages)
	h.router.POST("/messages", h.createMessage)
	h.router.PUT("/messages/:id", h.updateMessage)
	h.router.DELETE("/messages/:id", h.deleteMessage)
}

func (h *Handler) listMessages(w nethttp.ResponseWriter, req bunrouter.Request) error {
	messages, err := h.userViews.ListMessageViews(req.Context())
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("list messages: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	return bunrouter.JSON(w, messages)
}

func (h *Handler) createMessage(w nethttp.ResponseWriter, req bunrouter.Request) error {
	var body createMessageRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	message := &bunmodel.Message{
		ExternalID: body.ExternalID,
		SenderID:   body.SenderID,
		ReceiverID: body.ReceiverID,
		Subject:    body.Subject,
		Text:       body.Text,
		DateSent:   body.DateSent,
	}
	if err := h.messages.CreateMessage(req.Context(), message); err != nil {
		nethttp.Error(w, fmt.Sprintf("create message: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

	w.WriteHeader(nethttp.StatusCreated)
	return bunrouter.JSON(w, message)
}

func (h *Handler) updateMessage(w nethttp.ResponseWriter, req bunrouter.Request) error {
	id, err := pathInt64(req, "id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	var body createMessageRequest
	if err := yaml.NewDecoder(req.Body).Decode(&body); err != nil {
		nethttp.Error(w, fmt.Sprintf("decode body: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	message := &bunmodel.Message{
		ID:         id,
		ExternalID: body.ExternalID,
		SenderID:   body.SenderID,
		ReceiverID: body.ReceiverID,
		Subject:    body.Subject,
		Text:       body.Text,
		DateSent:   body.DateSent,
	}
	if err := h.messages.UpdateMessage(req.Context(), message); err != nil {
		nethttp.Error(w, fmt.Sprintf("update message: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

	return bunrouter.JSON(w, message)
}

func (h *Handler) deleteMessage(w nethttp.ResponseWriter, req bunrouter.Request) error {
	id, err := pathInt64(req, "id")
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("parse id: %v", err), nethttp.StatusBadRequest)
		return nil
	}

	_, err = h.messages.DeleteMessage(req.Context(), id)
	if err != nil {
		nethttp.Error(w, fmt.Sprintf("delete message: %v", err), nethttp.StatusInternalServerError)
		return nil
	}
	// Event captured by PostgreSQL trigger → domain_events → worker → Kafka

	w.WriteHeader(nethttp.StatusNoContent)
	return nil
}
