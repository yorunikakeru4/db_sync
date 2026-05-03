package seedinput

import (
	"github.com/uptrace/bun"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// Org represents an organization fixture input.
type Org struct {
	bun.BaseModel `bun:"orgs,alias:o"`

	Key string `yaml:"key" json:"key" bun:"-"`

	Name           *string           `yaml:"name" json:"name"`
	Status         *models.OrgStatus `yaml:"status" json:"status"`
	SubscriptionID *uint64           `yaml:"subscription_id" json:"subscriptionId" bun:",nullzero"`
	OnPremises     *bool             `yaml:"on_premises" json:"onPremises"`
	Budget         *float64          `yaml:"budget" json:"budget"`
	CreatedAt      unixtime.Nano     `yaml:"created_at" json:"createdAt"`
	UpdatedAt      unixtime.Nano     `yaml:"updated_at" json:"updatedAt"`
}

// FixtureKey returns the fixture key for this org.
func (o *Org) FixtureKey() string { return o.Key }

// Validate validates the org input.
func (o *Org) Validate() error {
	if o.Name != nil && *o.Name == "" {
		return validerr.Empty("name")
	}
	return nil
}

// OrgUser represents an org user membership fixture input.
type OrgUser struct {
	bun.BaseModel `bun:"org_users,alias:ou"`

	Key string `yaml:"key" json:"key" bun:"-"`

	OrgKey  string `yaml:"org_key" json:"orgKey" bun:"-"`
	OrgID   uint64 `yaml:"-" json:"-" bun:",nullzero"`
	UserKey string `yaml:"user_key" json:"userKey" bun:"-"`
	UserID  uint64 `yaml:"-" json:"-" bun:",nullzero"`

	Role *models.UserRole `yaml:"role" json:"role"`
}

// FixtureKey returns the fixture key for this org user.
func (ou *OrgUser) FixtureKey() string { return ou.Key }
