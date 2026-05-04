// Package middleware provides cross-cutting concerns for the event pipeline.
package middleware

import (
	"context"
	"log/slog"
	"time"

	"db_sync/internal/application/events"
	"db_sync/internal/service"
)

// EventConsumer is the interface for reading and dispatching Kafka events.
// KafkaConsumer from transport/kafka satisfies this interface.
type EventConsumer interface {
	GetEvent(ctx context.Context) (*events.Event, error)
	DispatchEvent(ctx context.Context, event *events.Event, syncService *service.SyncService) error
	Close() error
}

// LoggingConsumer wraps an EventConsumer and emits structured log lines
// for every received event (incoming) and every dispatch outcome (outgoing).
type LoggingConsumer struct {
	inner  EventConsumer
	logger *slog.Logger
}

// NewLoggingConsumer returns a LoggingConsumer that decorates inner with structured logging.
func NewLoggingConsumer(inner EventConsumer, logger *slog.Logger) *LoggingConsumer {
	return &LoggingConsumer{inner: inner, logger: logger}
}

// GetEvent reads the next event from the inner consumer and logs its receipt.
func (lc *LoggingConsumer) GetEvent(ctx context.Context) (*events.Event, error) {
	event, err := lc.inner.GetEvent(ctx)
	if err != nil {
		return nil, err
	}
	lc.logger.InfoContext(ctx, "event received",
		slog.String("event_id", event.EventID),
		slog.String("event_type", event.EventType),
		slog.Int("version", event.Version),
		slog.Time("emitted_at", event.Timestamp),
		slog.Any("payload", event.Payload),
	)
	return event, nil
}

// DispatchEvent routes the event to its handler, then logs the duration and outcome.
func (lc *LoggingConsumer) DispatchEvent(
	ctx context.Context,
	event *events.Event,
	syncService *service.SyncService,
) error {
	start := time.Now()
	lc.logger.InfoContext(ctx, "dispatching event",
		slog.String("event_id", event.EventID),
		slog.String("event_type", event.EventType),
		slog.Any("payload", event.Payload),
	)

	err := lc.inner.DispatchEvent(ctx, event, syncService)
	dur := time.Since(start)

	if err != nil {
		lc.logger.ErrorContext(ctx, "event dispatch failed",
			slog.String("event_id", event.EventID),
			slog.String("event_type", event.EventType),
			slog.Duration("duration", dur),
			slog.Any("error", err),
		)
		return err
	}

	lc.logger.InfoContext(ctx, "event dispatched",
		slog.String("event_id", event.EventID),
		slog.String("event_type", event.EventType),
		slog.Duration("duration", dur),
	)
	return nil
}

// Close delegates to the inner consumer.
func (lc *LoggingConsumer) Close() error {
	return lc.inner.Close()
}
