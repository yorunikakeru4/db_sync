// Package kafka provides event publishers for backend write-side mutations.
package kafka

import (
	"context"
	"encoding/json"
	"time"

	"backend/internal/events"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
)

// Producer publishes write-side events.
type Producer interface {
	// Publish emits a single event.
	Publish(ctx context.Context, eventType string, payload any) error
	// Close releases producer resources.
	Close() error
}

type kafkaProducer struct {
	writer *kafkago.Writer
}

// NewKafkaProducer creates a Kafka-backed producer for a single topic.
func NewKafkaProducer(brokers []string, topic string) Producer {
	return &kafkaProducer{
		writer: &kafkago.Writer{
			Addr:  kafkago.TCP(brokers...),
			Topic: topic,
		},
	}
}

// Publish marshals and writes an event to Kafka.
func (p *kafkaProducer) Publish(ctx context.Context, eventType string, payload any) error {
	body, err := json.Marshal(events.Event{
		EventID:   uuid.NewString(),
		EventType: eventType,
		Version:   1,
		Timestamp: time.Now().UTC(),
		Payload:   payload,
	})
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafkago.Message{Value: body})
}

// Close releases the underlying Kafka writer.
func (p *kafkaProducer) Close() error {
	return p.writer.Close()
}
