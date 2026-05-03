package seedinput

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// Dashboard represents a dashboard fixture input.
type Dashboard struct {
	bun.BaseModel `bun:"dashboards,alias:d"`

	Key string `yaml:"key" json:"key" bun:"-"`

	ProjectKey string `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32 `yaml:"-" json:"-" bun:",nullzero"`

	Name         *string  `yaml:"name" json:"name"`
	Pinned       bool     `yaml:"pinned" json:"pinned"`
	SortingOrder int64    `yaml:"sorting_order" json:"sortingOrder"`
	Tags         []string `yaml:"tags" json:"tags" bun:"type:varchar(100)[],nullzero"`

	Table *models.TableGridItemParams `yaml:"table" json:"table" bun:"type:jsonb"`

	GridVariables []string `yaml:"grid_variables" json:"gridVariables" bun:"type:jsonb,nullzero"`

	CreatedAt unixtime.Nano `yaml:"created_at" json:"createdAt"`
	UpdatedAt unixtime.Nano `yaml:"updated_at" json:"updatedAt"`
}

// FixtureKey returns the fixture key for this dashboard.
func (d *Dashboard) FixtureKey() string { return d.Key }

// Validate validates the dashboard input.
func (d *Dashboard) Validate() error {
	if d.Name != nil && *d.Name == "" {
		return validerr.Empty("name")
	}
	return nil
}

// ProjectTag represents a project tag fixture input.
type ProjectTag struct {
	bun.BaseModel `bun:"project_tags,alias:pt"`

	Key string `yaml:"key" json:"key" bun:"-"`

	ProjectKey string `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32 `yaml:"-" json:"-" bun:",nullzero"`

	Name         string `yaml:"name" json:"name"`
	SortingOrder int    `yaml:"sorting_order" json:"sortingOrder"`
}

// FixtureKey returns the fixture key for this project tag.
func (t *ProjectTag) FixtureKey() string { return t.Key }

// Validate validates the project tag input.
func (t *ProjectTag) Validate() error {
	if t.Name == "" {
		return validerr.Empty("name")
	}
	return nil
}

// TaggedDashboard represents a tagged dashboard association fixture input.
type TaggedDashboard struct {
	bun.BaseModel `bun:"tagged_dashboards,alias:td"`

	Key string `yaml:"key" json:"key" bun:"-"`

	TagKey       string `yaml:"tag_key" json:"tagKey" bun:"-"`
	TagID        uint64 `yaml:"-" json:"-"`
	DashboardKey string `yaml:"dashboard_key" json:"dashboardKey" bun:"-"`
	DashboardID  uint64 `yaml:"-" json:"-"`
	SortingOrder int    `yaml:"sorting_order" json:"sortingOrder"`
}

// FixtureKey returns the fixture key for this tagged dashboard.
func (td *TaggedDashboard) FixtureKey() string { return td.Key }
