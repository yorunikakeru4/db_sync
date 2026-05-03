package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"db_sync/internal/application/events"
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
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
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
	assert.Contains(t, logLine, `"msg":"event received"`)
	assert.Contains(t, logLine, `"event_id":"evt-abc-123"`)
	assert.Contains(t, logLine, `"event_type":"user_created"`)
	assert.Contains(t, logLine, `"version":1`)
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
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	require.Len(t, lines, 2, "expected two log lines: dispatching + dispatched")

	assert.Contains(t, lines[0], `"msg":"dispatching event"`)
	assert.Contains(t, lines[0], `"event_id":"evt-abc-123"`)

	assert.Contains(t, lines[1], `"msg":"event dispatched"`)
	assert.Contains(t, lines[1], `"duration"`)
	assert.Contains(t, lines[1], `"level":"INFO"`)
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
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	require.Len(t, lines, 2, "expected dispatching + error log lines")

	assert.Contains(t, lines[1], `"level":"ERROR"`)
	assert.Contains(t, lines[1], `"msg":"event dispatch failed"`)
	assert.Contains(t, lines[1], `"error":"mongo write failed"`)
	assert.Contains(t, lines[1], `"duration"`)
}

func TestLoggingConsumer_Close_Delegates(t *testing.T) {
	inner := &fakeConsumer{}
	lc := NewLoggingConsumer(inner, newTestLogger(&bytes.Buffer{}))

	err := lc.Close()

	require.NoError(t, err)
	assert.True(t, inner.closed, "Close must be delegated to inner consumer")
}
