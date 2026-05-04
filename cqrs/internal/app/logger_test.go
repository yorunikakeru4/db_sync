package app

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"db_sync/internal/logutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRuntimeLogger_FormatsStructuredText(t *testing.T) {
	var buf bytes.Buffer
	logger := logutil.NewRuntimeLogger(&buf)

	logger.InfoContext(
		context.Background(),
		"event dispatched",
		slog.String("event_id", "evt-123"),
		slog.Time("emitted_at", time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)),
		slog.Any("payload", map[string]any{"user_id": 7, "email": "readable@example.com"}),
	)

	line := strings.TrimSpace(buf.String())
	require.NotEmpty(t, line)
	assert.Contains(t, line, "time=")
	assert.Contains(t, line, "level=INFO")
	assert.Contains(t, line, "msg=\"event dispatched\"")
	assert.Contains(t, line, "event_id=evt-123")
	assert.Contains(t, line, "emitted_at=2026-05-04T12:00:00Z")
	assert.Contains(t, line, "payload=\"email=readable@example.com, user_id=7\"")
	assert.NotContains(t, line, "{\"")
}
