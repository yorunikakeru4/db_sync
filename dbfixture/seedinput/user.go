package seedinput

import (
	"strings"

	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// User represents a user fixture input.
type User struct {
	bun.BaseModel `bun:"users,alias:u"`

	Key string `yaml:"key" json:"key" bun:"-"`

	Name           *string       `yaml:"name" json:"name"`
	Email          *string       `yaml:"email" json:"email"`
	EmailConfirmed *bool         `yaml:"email_confirmed" json:"-"`
	Password       *string       `yaml:"password" json:"password"`
	Avatar         *string       `yaml:"avatar" json:"avatar"`
	CreatedAt      unixtime.Nano `yaml:"created_at" json:"createdAt"`

	Orgs     []UserOrgMembership     `yaml:"orgs" json:"orgs" bun:"-"`
	Projects []UserProjectMembership `yaml:"projects" json:"projects" bun:"-"`
}

// UserOrgMembership represents a user's membership in an org.
type UserOrgMembership struct {
	OrgKey string           `yaml:"org_key" json:"orgKey"`
	Role   *models.UserRole `yaml:"role" json:"role"`
}

// UserProjectMembership represents a user's membership in a project.
type UserProjectMembership struct {
	ProjectKey string            `yaml:"project_key" json:"projectKey"`
	PermLevel  *models.PermLevel `yaml:"perm_level" json:"permLevel"`
}

// FixtureKey returns the fixture key for this user.
func (u *User) FixtureKey() string { return u.Key }

// Validate validates the user input.
func (u *User) Validate() error {
	if u.Name != nil && *u.Name == "" {
		return validerr.Empty("name")
	}
	if u.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*u.Email))
		if email == "" {
			return validerr.Empty("email")
		}
		u.Email = &email
	}
	if u.Avatar != nil && *u.Avatar == "" {
		return validerr.Empty("avatar")
	}
	return nil
}

// UserToken represents a user token fixture input.
type UserToken struct {
	bun.BaseModel `bun:"user_tokens,alias:ut"`

	Key string `yaml:"key" json:"key" bun:"-"`

	UserKey string `yaml:"user_key" json:"userKey" bun:"-"`
	UserID  uint64 `yaml:"-" json:"-" bun:",nullzero"`

	Token     string        `yaml:"token" json:"token"`
	CreatedAt unixtime.Nano `yaml:"created_at" json:"createdAt"`
}

// FixtureKey returns the fixture key for this user token.
func (t *UserToken) FixtureKey() string { return t.Key }

// Validate validates the user token input.
func (t *UserToken) Validate() error {
	if t.Key == "" {
		return validerr.Empty("key")
	}
	if t.Token == "" {
		return validerr.Empty("token")
	}
	return nil
}
