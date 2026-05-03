// Package events defines Kafka event envelopes and payloads emitted by the backend.
package events

import "time"

// Event is the generic Kafka message envelope.
type Event struct {
	// EventID is the unique identifier of the event.
	EventID string `json:"event_id"`
	// EventType is the snake_case event discriminator.
	EventType string `json:"event_type"`
	// Version is the schema version of the event payload.
	Version int `json:"version"`
	// Timestamp is the UTC time the event was emitted.
	Timestamp time.Time `json:"timestamp"`
	// Payload carries the event-specific data.
	Payload any `json:"payload"`
}
