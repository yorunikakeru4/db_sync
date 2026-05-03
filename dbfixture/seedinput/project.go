package seedinput

import (
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// Project represents a project fixture input.
type Project struct {
	bun.BaseModel `bun:"projects,alias:p"`

	Key string `yaml:"key" json:"key" bun:"-"`

	OrgKey string `yaml:"org_key" json:"orgKey" bun:"-"`
	OrgID  uint64 `yaml:"-" json:"-" bun:",nullzero"`

	Name      *string `yaml:"name" json:"name"`
	Suspended *bool   `yaml:"suspended" json:"-"`

	SpanQuery     *string        `yaml:"span_query" json:"spanQuery"`
	SpanColumns   *[]string      `yaml:"span_columns" json:"spanColumns" bun:",array"`
	SpanRetention *time.Duration `yaml:"span_retention" json:"spanRetention" bun:",nullzero"`

	LogQuery     *string        `yaml:"log_query" json:"logQuery"`
	LogColumns   *[]string      `yaml:"log_columns" json:"logColumns" bun:",array"`
	LogRetention *time.Duration `yaml:"log_retention" json:"logRetention" bun:",nullzero"`

	EventQuery     *string        `yaml:"event_query" json:"eventQuery"`
	EventColumns   *[]string      `yaml:"event_columns" json:"eventColumns" bun:",array"`
	EventRetention *time.Duration `yaml:"event_retention" json:"eventRetention" bun:",nullzero"`

	TraceQuery   *string   `yaml:"trace_query" json:"traceQuery"`
	TraceColumns *[]string `yaml:"trace_columns" json:"traceColumns" bun:",array"`

	MetricRetention *time.Duration `yaml:"metric_retention" json:"metricRetention" bun:",nullzero"`

	CreatedAt unixtime.Nano `yaml:"created_at" json:"createdAt"`
	UpdatedAt unixtime.Nano `yaml:"updated_at" json:"updatedAt"`
}

// FixtureKey returns the fixture key for this project.
func (p *Project) FixtureKey() string { return p.Key }

// Validate validates the project input.
func (p *Project) Validate() error {
	if p.Name != nil && *p.Name == "" {
		return validerr.Empty("name")
	}
	return nil
}

// ProjectToken represents a project token fixture input.
type ProjectToken struct {
	bun.BaseModel `bun:"project_tokens,alias:pt"`

	Key string `yaml:"key" json:"key" bun:"-"`

	ProjectKey string `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32 `yaml:"-" json:"-" bun:",nullzero"`

	Token     string        `yaml:"token" json:"token"`
	CreatedAt unixtime.Nano `yaml:"created_at" json:"createdAt"`
}

// FixtureKey returns the fixture key for this project token.
func (pt *ProjectToken) FixtureKey() string { return pt.Key }

// ProjectUser represents a project user membership fixture input.
type ProjectUser struct {
	bun.BaseModel `bun:"org_user_projects,alias:oup"`

	Key string `yaml:"key" json:"key" bun:"-"`

	OrgID      uint64 `yaml:"-" json:"-"`
	ProjectKey string `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32 `yaml:"-" json:"-" bun:",nullzero"`
	OrgUserKey string `yaml:"org_user_key" json:"orgUserKey" bun:"-"`
	OrgUserID  uint64 `yaml:"-" json:"-" bun:",nullzero"`

	PermLevel *models.PermLevel `yaml:"perm_level" json:"permLevel"`
}

// FixtureKey returns the fixture key for this project user.
func (pu *ProjectUser) FixtureKey() string { return pu.Key }
