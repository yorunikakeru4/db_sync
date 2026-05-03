// Package events defines the domain event envelope and payload types consumed from Kafka.
package events

import "time"

// Event is the generic Kafka message envelope. The Payload field is decoded
// into a concrete payload type based on EventType.
type Event struct {
	// EventID is the unique identifier of the event.
	EventID string `json:"event_id"`
	// EventType is the snake_case event discriminator (e.g. user_created).
	EventType string `json:"event_type"`
	// Version is the schema version of the event payload.
	Version int `json:"version"`
	// Timestamp is the UTC time the event was emitted.
	Timestamp time.Time `json:"timestamp"`
	// Payload carries the event-specific data; decoded with mapstructure.
	Payload any `json:"payload"`
}
