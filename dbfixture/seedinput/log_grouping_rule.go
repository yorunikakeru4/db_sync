package seedinput

// LogGroupingRule represents a log grouping rule fixture input.
type LogGroupingRule struct {
	Key        string   `yaml:"key" json:"key"`
	ProjectKey string   `yaml:"project_key" json:"projectKey"`
	ProjectID  uint32   `yaml:"-" json:"-"`
	Patterns   []string `yaml:"patterns" json:"patterns"`
	// Status is the initial status of the rule (active, paused, failed).
	// When empty, the database default (active) is used.
	Status string `yaml:"status,omitempty" json:"status,omitempty"`
	// Error is the error text associated with a failed rule.
	Error string `yaml:"error,omitempty" json:"error,omitempty"`
}

// FixtureKey returns the fixture key for this log grouping rule.
func (r *LogGroupingRule) FixtureKey() string { return r.Key }
