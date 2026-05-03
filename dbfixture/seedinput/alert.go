package seedinput

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
)

// BaseAlert represents the base alert fixture input.
type BaseAlert struct {
	bun.BaseModel `bun:"alerts,alias:a"`

	Key string `yaml:"key" json:"key" bun:"-"`

	ProjectKey string `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32 `yaml:"-" json:"-" bun:",nullzero"`
	MonitorKey string `yaml:"monitor_key" bun:"-"`
	MonitorID  uint64 `yaml:"-" json:"-" bun:",nullzero"`

	Name      string           `yaml:"name"`
	Type      models.AlertType `yaml:"-"`
	Attrs     map[string]any   `yaml:"attrs"`
	AttrsHash uint64           `yaml:"attrs_hash"`

	HourlyTrend    []float64               `yaml:"hourly_trend" bun:",array"`
	TrendUpdatedAt unixtime.Nano           `yaml:"trend_updated_at"`
	TrendAggFunc   models.TrendAggFuncName `yaml:"trend_agg_func"`

	CreatedAt unixtime.Nano `yaml:"created_at"`
	UpdatedAt unixtime.Nano `yaml:"updated_at"`
}

// FixtureKey returns the fixture key for this alert.
func (a *BaseAlert) FixtureKey() string { return a.Key }

// MetricAlert represents a metric alert fixture input.
type MetricAlert struct {
	BaseAlert `yaml:",inline" bun:",inherit"`
}

// ErrorAlert represents an error alert fixture input.
type ErrorAlert struct {
	BaseAlert `yaml:",inline" bun:",inherit"`
}

// AlertEvent is implemented by alert event types that can be inserted.
type AlertEvent interface {
	SetAlertID(id uint64)
	SetProjectID(id uint32)
}

// BaseAlertEvent represents the base alert event fixture input.
type BaseAlertEvent struct {
	bun.BaseModel `bun:"alert_events,alias:e"`

	AlertID   uint64 `yaml:"-"`
	ProjectID uint32 `yaml:"-" json:"-" bun:",nullzero"`

	Name         models.AlertEventName `yaml:"name"`
	SerialNumber int                   `yaml:"serial_number"`

	Status   models.AlertStatus   `yaml:"status"`
	Priority models.AlertPriority `yaml:"priority"`

	ActivityState  models.AlertActivityState `yaml:"activity_state"`
	StateUpdatedAt unixtime.Nano             `yaml:"state_updated_at"`

	ArchivedUntil unixtime.Nano `yaml:"archived_until"`

	CurrentValue float64       `yaml:"current_value"`
	ValueTime    unixtime.Nano `yaml:"value_time"`

	CreatedAt unixtime.Nano `yaml:"created_at"`
}

// SetAlertID sets the alert ID on the event.
func (e *BaseAlertEvent) SetAlertID(id uint64) { e.AlertID = id }

// SetProjectID sets the project ID on the event.
func (e *BaseAlertEvent) SetProjectID(id uint32) { e.ProjectID = id }

// MetricAlertEvent represents a metric alert event fixture input.
type MetricAlertEvent struct {
	bun.BaseModel  `bun:"alert_events,alias:e"`
	BaseAlertEvent `yaml:",inline" bun:",inherit"`

	Key      string `yaml:"key" json:"key" bun:"-"`
	AlertKey string `yaml:"alert_key" bun:"-"`

	Params models.MetricAlertParams `yaml:"params" bun:"type:jsonb,nullzero"`
}

// FixtureKey returns the fixture key for this metric alert event.
func (e *MetricAlertEvent) FixtureKey() string { return e.Key }

// ErrorAlertEvent represents an error alert event fixture input.
type ErrorAlertEvent struct {
	bun.BaseModel  `bun:"alert_events,alias:e"`
	BaseAlertEvent `yaml:",inline" bun:",inherit"`

	Key      string `yaml:"key" json:"key" bun:"-"`
	AlertKey string `yaml:"alert_key" bun:"-"`

	Params models.ErrorAlertParams `yaml:"params" bun:"type:jsonb,nullzero"`
}

// FixtureKey returns the fixture key for this error alert event.
func (e *ErrorAlertEvent) FixtureKey() string { return e.Key }

// AlertNote represents an alert note fixture input.
type AlertNote struct {
	Key string `yaml:"key" json:"key"`

	ProjectKey string `yaml:"project_key" json:"projectKey"`
	ProjectID  uint32 `yaml:"-" json:"-"`
	UserKey    string `yaml:"user_key"`
	UserID     uint64 `yaml:"-" json:"-"`
	AlertKey   string `yaml:"alert_key"`
	AlertID    uint64 `yaml:"-" json:"-"`

	Text *string `yaml:"text"`

	CreatedAt unixtime.Nano `yaml:"created_at"`
}

// FixtureKey returns the fixture key for this alert note.
func (n *AlertNote) FixtureKey() string { return n.Key }

// NewEvent creates a new NoteAlertEvent from the alert note.
func (n *AlertNote) NewEvent() *models.NoteAlertEvent {
	base := &models.BaseAlertEvent{
		AlertID:   n.AlertID,
		ProjectID: n.ProjectID,
		UserID:    n.UserID,
	}
	params := models.NoteAlertEventParams{}
	if n.Text != nil {
		params.Text = *n.Text
	}
	event := models.NewNoteAlertEvent(base, params)
	if !n.CreatedAt.IsZero() {
		event.CreatedAt = n.CreatedAt
	}
	if event.Status == "" {
		event.Status = models.AlertStatusUnresolved
	}
	if event.ActivityState == "" {
		event.ActivityState = models.AlertActivityNew
	}
	if event.Priority == "" {
		event.Priority = models.AlertPriorityLow
	}
	return event
}

// AlertAssignment represents an alert assignment fixture input.
type AlertAssignment struct {
	bun.BaseModel `bun:"alert_assignments,alias:aa"`

	Key string `yaml:"key" json:"key" bun:"-"`

	AlertKey string `yaml:"alert_key" bun:"-"`
	AlertID  uint64 `yaml:"-" json:"-" bun:",nullzero"`
	UserKey  string `yaml:"user_key" bun:"-"`
	UserID   uint64 `yaml:"-" json:"-" bun:",nullzero"`
}

// FixtureKey returns the fixture key for this alert assignment.
func (a *AlertAssignment) FixtureKey() string { return a.Key }
