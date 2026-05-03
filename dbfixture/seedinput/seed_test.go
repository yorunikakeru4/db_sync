package seedinput

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSpanNewTraceID_Uint64Attrs(t *testing.T) {
	t.Parallel()

	span := &Span{
		TraceIDSeed: 42,
		Attrs: map[string]any{
			"_trace_id_low":  uint64(1),
			"_trace_id_high": uint64(3),
		},
	}

	traceID := span.NewTraceID()

	require.Equal(t, uint64(1), traceID.Low())
	require.Equal(t, uint64(3), traceID.High())
	require.Empty(t, span.Attrs)
}
