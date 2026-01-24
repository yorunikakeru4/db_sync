package events

import "time"

type Event struct {
	EventID   string    `json:"event_id"`
	EventType string    `json:"event_type"`
	Version   int       `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Payload   any       `json:"payload"`
}
