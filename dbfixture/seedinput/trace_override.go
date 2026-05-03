package seedinput

import (
	"github.com/uptrace/uptrace/pkg/idgen"
	"github.com/uptrace/uptrace/pkg/unixtime"
)

// TraceOverride represents a trace override fixture input.
type TraceOverride struct {
	Key        string        `yaml:"key"`
	ProjectKey string        `yaml:"project_key"`
	TraceID    idgen.TraceID `yaml:"trace_id"`
	ExpiresAt  unixtime.Nano `yaml:"expires_at"`
	UserKey    string        `yaml:"user_key"`
	UserID     uint64        `yaml:"-"`
	Reason     *string       `yaml:"reason"`
}

// FixtureKey returns the fixture key for this trace override.
func (t *TraceOverride) FixtureKey() string { return t.Key }
