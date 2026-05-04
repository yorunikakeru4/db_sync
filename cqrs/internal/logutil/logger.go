package logutil

import (
	"fmt"
	"io"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"time"
)

// NewRuntimeLogger builds the shared human-readable logger for the CQRS runtime.
func NewRuntimeLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: replaceRuntimeAttr,
	}))
}

func replaceRuntimeAttr(_ []string, attr slog.Attr) slog.Attr {
	switch attr.Key {
	case slog.TimeKey, "emitted_at":
		if attr.Value.Kind() == slog.KindTime {
			return slog.String(attr.Key, attr.Value.Time().UTC().Format(time.RFC3339))
		}
	case slog.LevelKey:
		return slog.String(attr.Key, strings.ToUpper(attr.Value.Any().(slog.Level).String()))
	case "payload":
		return slog.String(attr.Key, formatValue(attr.Value.Any()))
	}
	return attr
}

func formatValue(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		return formatMap(typed)
	case nil:
		return ""
	default:
		return fmt.Sprint(value)
	}
}

func formatMap(values map[string]any) string {
	if len(values) == 0 {
		return ""
	}

	keys := slices.Sorted(maps.Keys(values))
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", key, values[key]))
	}
	return strings.Join(parts, ", ")
}
