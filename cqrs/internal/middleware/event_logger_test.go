package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"db_sync/internal/application/events"
	"db_sync/internal/logutil"
	"db_sync/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeConsumer is a hand-rolled EventConsumer stub for middleware tests.
type fakeConsumer struct {
	event   *events.Event
	getErr  error
	dispErr error
	closed  bool
}

func (f *fakeConsumer) GetEvent(_ context.Context) (*events.Event, error) {
	return f.event, f.getErr
}

func (f *fakeConsumer) DispatchEvent(_ context.Context, _ *events.Event, _ *service.SyncService) error {
	return f.dispErr
}

func (f *fakeConsumer) Close() error {
	f.closed = true
	return nil
}

func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return logutil.NewRuntimeLogger(buf)
}

func sampleEvent() *events.Event {
	return &events.Event{
		EventID:   "evt-abc-123",
		EventType: "user_created",
		Version:   1,
		Timestamp: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

func TestLoggingConsumer_GetEvent_Success(t *testing.T) {
	var buf bytes.Buffer
	inner := &fakeConsumer{event: sampleEvent()}
	lc := NewLoggingConsumer(inner, newTestLogger(&buf))

	got, err := lc.GetEvent(context.Background())

	require.NoError(t, err)
	assert.Equal(t, sampleEvent(), got)

	logLine := buf.String()
	assert.Contains(t, logLine, "msg=\"event received\"")
	assert.Contains(t, logLine, "event_id=evt-abc-123")
	assert.Contains(t, logLine, "event_type=user_created")
	assert.Contains(t, logLine, "version=1")
	assert.Contains(t, logLine, "emitted_at=2024-01-15T10:00:00Z")
}

func TestLoggingConsumer_GetEvent_Error(t *testing.T) {
	var buf bytes.Buffer
	inner := &fakeConsumer{getErr: errors.New("kafka timeout")}
	lc := NewLoggingConsumer(inner, newTestLogger(&buf))

	got, err := lc.GetEvent(context.Background())

	require.Error(t, err)
	assert.Nil(t, got)
	assert.Empty(t, buf.String(), "no log line on error")
}

func TestLoggingConsumer_DispatchEvent_Success(t *testing.T) {
	var buf bytes.Buffer
	inner := &fakeConsumer{dispErr: nil}
	lc := NewLoggingConsumer(inner, newTestLogger(&buf))

	err := lc.DispatchEvent(context.Background(), sampleEvent(), nil)

	require.NoError(t, err)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "msg=\"dispatching event\"")
	assert.Contains(t, logOutput, "msg=\"event dispatched\"")
	assert.Contains(t, logOutput, "event_id=evt-abc-123")
	assert.Contains(t, logOutput, "level=INFO")
	assert.Contains(t, logOutput, "duration=")
}

func TestLoggingConsumer_DispatchEvent_Error(t *testing.T) {
	var buf bytes.Buffer
	dispErr := errors.New("mongo write failed")
	inner := &fakeConsumer{dispErr: dispErr}
	lc := NewLoggingConsumer(inner, newTestLogger(&buf))

	err := lc.DispatchEvent(context.Background(), sampleEvent(), nil)

	require.Error(t, err)
	assert.Equal(t, dispErr, err)

	logOutput := buf.String()
	assert.Contains(t, logOutput, "level=ERROR")
	assert.Contains(t, logOutput, "msg=\"event dispatch failed\"")
	assert.Contains(t, logOutput, "error=\"mongo write failed\"")
	assert.Contains(t, logOutput, "duration=")
}

func TestLoggingConsumer_Close_Delegates(t *testing.T) {
	inner := &fakeConsumer{}
	lc := NewLoggingConsumer(inner, newTestLogger(&bytes.Buffer{}))

	err := lc.Close()

	require.NoError(t, err)
	assert.True(t, inner.closed, "Close must be delegated to inner consumer")
}
