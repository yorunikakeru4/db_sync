package kafka

import (
	"context"
	"sync"
	"time"

	"backend/internal/events"
)

// MemProducer stores published events in memory for tests.
type MemProducer struct {
	// Events holds published events in order.
	Events []events.Event

	mu sync.Mutex
}

// Publish records an event in memory.
func (p *MemProducer) Publish(_ context.Context, eventType string, payload any) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Events = append(p.Events, events.Event{
		EventType: eventType,
		Version:   1,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})

	return nil
}

// Close is a no-op for the in-memory producer.
func (p *MemProducer) Close() error {
	return nil
}
