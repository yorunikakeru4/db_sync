// Package eventworker polls domain_events and publishes to Kafka.
package eventworker

import (
	"context"
	"encoding/json"
	"time"

	"backend/internal/kafka"
)

// DomainEvent represents a row from the domain_events table.
type DomainEvent struct {
	// ID is the primary key.
	ID int64 `bun:"id,pk,autoincrement"`
	// EventType is the event discriminator (e.g. "user_created").
	EventType string `bun:"event_type"`
	// AggregateType is the domain aggregate (e.g. "user", "message").
	AggregateType string `bun:"aggregate_type"`
	// AggregateID is the primary key of the aggregate.
	AggregateID int64 `bun:"aggregate_id"`
	// Payload is the event-specific data as JSON.
	Payload json.RawMessage `bun:"payload"`
	// CreatedAt is when the event was captured.
	CreatedAt time.Time `bun:"created_at"`
	// PublishedAt is when the event was published to Kafka (NULL if unpublished).
	PublishedAt *time.Time `bun:"published_at"`
}

// Repository fetches and updates domain events.
type Repository interface {
	// FetchUnpublished returns events not yet published, ordered by ID.
	FetchUnpublished(ctx context.Context, limit int) ([]DomainEvent, error)
	// MarkPublished sets published_at for the given event IDs.
	MarkPublished(ctx context.Context, ids []int64) error
}

// Worker polls domain_events and publishes to Kafka.
type Worker struct {
	repo         Repository
	producer     kafka.Producer
	pollInterval time.Duration
	batchSize    int
}

// Option configures the worker.
type Option func(*Worker)

// WithPollInterval sets the polling interval.
func WithPollInterval(d time.Duration) Option {
	return func(w *Worker) {
		w.pollInterval = d
	}
}

// WithBatchSize sets the maximum events per poll cycle.
func WithBatchSize(n int) Option {
	return func(w *Worker) {
		w.batchSize = n
	}
}

// NewWorker creates a worker with the given options.
func NewWorker(repo Repository, producer kafka.Producer, opts ...Option) *Worker {
	w := &Worker{
		repo:         repo,
		producer:     producer,
		pollInterval: 100 * time.Millisecond,
		batchSize:    100,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Run starts the polling loop. Blocks until context is canceled.
func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			_ = w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) error {
	events, err := w.repo.FetchUnpublished(ctx, w.batchSize)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	publishedIDs := make([]int64, 0, len(events))
	for _, evt := range events {
		var payload any
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			continue
		}

		if err := w.producer.Publish(ctx, evt.EventType, payload); err != nil {
			continue
		}
		publishedIDs = append(publishedIDs, evt.ID)
	}

	if len(publishedIDs) > 0 {
		if err := w.repo.MarkPublished(ctx, publishedIDs); err != nil {
			return err
		}
	}

	return nil
}
