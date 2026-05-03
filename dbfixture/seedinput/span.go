package seedinput

import (
	"strconv"

	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/idgen"
	"github.com/uptrace/uptrace/pkg/unixtime"
)

// Span represents a span fixture input.
// Used for spans, logs, and events (distinguished by Type field).
type Span struct {
	ProjectKey string `yaml:"project_key"`
	ProjectID  uint32 `yaml:"-" json:"-"`

	ID          idgen.SpanID  `yaml:"id"`
	TraceID     idgen.TraceID `yaml:"trace_id"`
	TraceIDSeed uint64        `yaml:"-"`
	ParentID    idgen.SpanID  `yaml:"parent_id"`

	Name      string              `yaml:"name"`
	Type      string              `yaml:"type"`
	System    string              `yaml:"system"`
	CountX100 uint32              `yaml:"count_x100"`
	GroupID   uint64              `yaml:"group_id"`
	Attrs     map[string]any      `yaml:"attrs"`
	Duration  models.Microseconds `yaml:"duration"`
	Size      uint32              `yaml:"size"`
	StartTime unixtime.Nano       `yaml:"start_time,omitempty"`
	Links     []SpanLink          `yaml:"links"`

	TextIndex string `yaml:"text_index"`
}

// FixtureKey returns the span ID as a string key.
func (s *Span) FixtureKey() string {
	return strconv.FormatUint(uint64(s.ID), 10)
}

// NewTraceID generates a new trace ID for the span.
func (s *Span) NewTraceID() idgen.TraceID {
	attrSeed := func(key string) (uint64, bool) {
		val, ok := s.Attrs[key]
		if !ok {
			return 0, false
		}
		delete(s.Attrs, key)

		switch v := val.(type) {
		case int:
			return uint64(v), true
		case int64:
			return uint64(v), true
		case uint64:
			return v, true
		case float64:
			return uint64(v), true
		default:
			return 0, false
		}
	}

	low, high := s.TraceIDSeed, s.TraceIDSeed
	if seed, ok := attrSeed("_trace_id"); ok {
		low = seed
		high = seed
	}
	if seed, ok := attrSeed("_trace_id_low"); ok {
		low = seed
	}
	if seed, ok := attrSeed("_trace_id_high"); ok {
		high = seed
	}

	if low != 0 || high != 0 {
		return idgen.NewTraceIDLowHigh(low, high)
	}
	return idgen.RandTraceID()
}

// SpanLink represents a span link fixture input.
// SpanID points to the linked span in the same or another trace.
// Attrs are link-specific attributes (not the linked span's attrs).
type SpanLink struct {
	SpanID idgen.SpanID   `yaml:"span_id"`
	Attrs  map[string]any `yaml:"attrs"`
}
