package seedinput

import (
	"reflect"
	"slices"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/ast"

	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/jsontype"
	"github.com/uptrace/uptrace/pkg/madalarm"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
	"github.com/uptrace/uptrace/tsmetric/mql"
)

// BaseMonitor represents the base monitor fixture input.
type BaseMonitor struct {
	bun.BaseModel `yaml:"-" bun:"monitors,alias:m"`

	Key string `yaml:"key" json:"key" bun:"key"`

	ProjectKey string   `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32   `yaml:"-" json:"-" bun:",nullzero"`
	TeamKeys   []string `yaml:"team_keys" json:"teamKeys" bun:"-"`

	Name            *string               `yaml:"name" json:"name"`
	IsNameTemplated bool                  `yaml:"is_name_templated" json:"isNameTemplated"`
	Type            models.MonitorType    `yaml:"-" json:"-"`
	Status          *models.MonitorStatus `yaml:"status" json:"status"`

	TrendAggFunc     *models.TrendAggFuncName `yaml:"trend_agg_func" json:"trendAggFunc"`
	TrendSensitivity *models.TrendSensitivity `yaml:"trend_sensitivity" json:"trendSensitivity"`

	RepeatInterval       *models.RepeatInterval       `yaml:"-" json:"-"`
	RepeatIntervalParams *models.RepeatIntervalParams `yaml:"repeat_interval" json:"-" bun:"-"`

	NotifyEveryoneByEmail *bool `yaml:"notify_everyone_by_email" json:"notifyEveryoneByEmail"`

	CreatedAt unixtime.Nano `yaml:"created_at" json:"createdAt" bun:",nullzero"`
	UpdatedAt unixtime.Nano `yaml:"updated_at" json:"updatedAt" bun:",nullzero"`
}

// FixtureKey returns the fixture key for this monitor.
func (m *BaseMonitor) FixtureKey() string { return m.Key }

// Validate validates the base monitor input.
func (m *BaseMonitor) Validate() error {
	if m.Name != nil && *m.Name == "" {
		return validerr.Empty("name")
	}
	if m.Status != nil && *m.Status == "" {
		return validerr.Empty("status")
	}
	if m.TrendAggFunc != nil && *m.TrendAggFunc == "" {
		return validerr.Empty("trend_agg_func")
	}
	return nil
}

// MetricMonitor represents a metric monitor fixture input.
type MetricMonitor struct {
	BaseMonitor `yaml:",inline" bun:",inherit"`

	Params *MetricMonitorParams `yaml:",inline" bun:",nullzero"`
}

// MetricMonitorParams holds metric monitor parameters.
type MetricMonitorParams struct {
	Metrics     []mql.MetricAlias `yaml:"-" json:"metrics"`
	MetricExprs []string          `yaml:"metrics" json:"-"` // used only for YAML decoding

	Query      string   `yaml:"-" json:"query"`
	QueryParts []string `yaml:"query" json:"-"` // used only for YAML decoding

	Column     string `yaml:"column"`
	ColumnUnit string `yaml:"column_unit,omitempty"`

	Resolution    time.Duration         `yaml:"resolution"`
	NumEvalPoints int                   `yaml:"num_eval_points"`
	AbsentPoints  madalarm.AbsentPoints `yaml:"absent_points" json:"absentPoints"`
	TimeOffset    time.Duration         `yaml:"time_offset,omitempty"`

	// Detector holds the detector type and its type-specific parameters.
	Detector madalarm.DetectorTpl `yaml:"detector"`

	// Deprecated: use Detector instead.
	BoundsSource    madalarm.BoundsSource `yaml:"bounds_source"`
	MinAllowedValue jsontype.NullFloat64  `yaml:"min_allowed_value,omitempty"`
	MaxAllowedValue jsontype.NullFloat64  `yaml:"max_allowed_value,omitempty"`
	Flapping        models.FlappingParams `yaml:"flapping"`
	Tolerance       madalarm.Tolerance    `yaml:"tolerance,omitempty"`
	TrainingPeriod  time.Duration         `yaml:"training_period,omitempty"`
	MinDevFraction  float64               `yaml:"min_dev_fraction,omitempty"`
	MinDevAbsolute  float64               `yaml:"min_dev_absolute,omitempty"`
}

// Validate validates the metric monitor input.
func (m *MetricMonitor) Validate() error {
	if err := m.BaseMonitor.Validate(); err != nil {
		return err
	}
	if m.Params == nil || len(m.Params.QueryParts) == 0 {
		return validerr.AtLeastOne("query")
	}
	if slices.Contains(m.Params.QueryParts, "") {
		return validerr.Empty("query")
	}
	if len(m.Params.MetricExprs) == 0 {
		return validerr.AtLeastOne("metrics")
	}
	if slices.Contains(m.Params.MetricExprs, "") {
		return validerr.Empty("metrics")
	}
	if m.Params.TimeOffset > 300*time.Minute {
		return validerr.TooLarge("time_offset", m.Params.TimeOffset, "300m")
	}
	return nil
}

var _ yaml.NodeUnmarshaler = (*MetricMonitor)(nil)

// UnmarshalYAML implements yaml.NodeUnmarshaler.
func (m *MetricMonitor) UnmarshalYAML(node ast.Node) error {
	type raw MetricMonitor
	if err := yaml.NodeToValue(node, (*raw)(m)); err != nil {
		return err
	}
	if m.Params != nil && reflect.DeepEqual(*m.Params, MetricMonitorParams{}) {
		m.Params = nil
	}
	return nil
}

// ErrorMonitor represents an error monitor fixture input.
type ErrorMonitor struct {
	BaseMonitor `yaml:",inline" bun:",inherit"`

	Params *ErrorMonitorParams `yaml:"params,inline" json:"params"`
}

// ErrorMonitorParams holds error monitor parameters.
type ErrorMonitorParams struct {
	Metrics []mql.MetricAlias `yaml:"-" json:"metrics"`

	QueryParts []string `yaml:"query" json:"-"`
	Query      string   `yaml:"-" json:"query"`
}

// Validate validates the error monitor input.
func (m *ErrorMonitor) Validate() error {
	if err := m.BaseMonitor.Validate(); err != nil {
		return err
	}
	if m.Params == nil || len(m.Params.QueryParts) == 0 {
		return validerr.AtLeastOne("query")
	}
	if slices.Contains(m.Params.QueryParts, "") {
		return validerr.Empty("query")
	}
	return nil
}

var _ yaml.NodeUnmarshaler = (*ErrorMonitor)(nil)

// UnmarshalYAML implements yaml.NodeUnmarshaler.
func (m *ErrorMonitor) UnmarshalYAML(node ast.Node) error {
	type raw ErrorMonitor
	if err := yaml.NodeToValue(node, (*raw)(m)); err != nil {
		return err
	}
	if m.Params != nil && reflect.DeepEqual(*m.Params, ErrorMonitorParams{}) {
		m.Params = nil
	}
	return nil
}
