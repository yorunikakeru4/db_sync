package seedinput

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// Team represents a team fixture input.
type Team struct {
	bun.BaseModel `bun:"teams,alias:t"`

	Key string `yaml:"key" json:"key" bun:"-"`

	OrgKey string `yaml:"org_key" json:"orgKey" bun:"-"`
	OrgID  uint64 `yaml:"-" json:"-" bun:",nullzero"`

	Name      *string           `yaml:"name" json:"name"`
	PermLevel *models.PermLevel `yaml:"perm_level" json:"permLevel"`
	CreatedAt unixtime.Nano     `yaml:"created_at" json:"createdAt"`
}

// FixtureKey returns the fixture key for this team.
func (t *Team) FixtureKey() string { return t.Key }

// Validate validates the team input.
func (t *Team) Validate() error {
	if t.Name != nil && *t.Name == "" {
		return validerr.Empty("name")
	}
	return nil
}

// TeamUser represents a team user membership fixture input.
type TeamUser struct {
	bun.BaseModel `bun:"alias:tu"`

	Key string `yaml:"key" json:"key" bun:"-"`

	TeamKey    string `yaml:"team_key" json:"teamKey" bun:"-"`
	TeamID     uint64 `yaml:"-" json:"-" bun:",nullzero"`
	OrgUserKey string `yaml:"org_user_key" json:"orgUserKey" bun:"-"`
	OrgUserID  uint64 `yaml:"-" json:"-" bun:",nullzero"`
}

// FixtureKey returns the fixture key for this team user.
func (tu *TeamUser) FixtureKey() string { return tu.Key }

// TeamProject represents a team project association fixture input.
type TeamProject struct {
	bun.BaseModel `bun:"alias:tp"`

	Key string `yaml:"key" json:"key" bun:"-"`

	TeamKey    string `yaml:"team_key" json:"teamKey" bun:"-"`
	TeamID     uint64 `json:"teamId"`
	ProjectKey string `yaml:"project_key" json:"projectKey" bun:"-"`
	ProjectID  uint32 `json:"projectId"`
}

// FixtureKey returns the fixture key for this team project.
func (tp *TeamProject) FixtureKey() string { return tp.Key }
