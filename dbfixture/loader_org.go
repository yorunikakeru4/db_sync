package dbfixture

import (
	"context"
	"math/rand/v2"

	"github.com/brianvoe/gofakeit/v5"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// OrgLoader loads organization fixtures.
type OrgLoader struct {
	pgLoaderBase[seedinput.Org, models.Org]
}

// NewOrgLoader creates a Loader for organizations.
func NewOrgLoader(f *Fixture, params LoaderParams) *OrgLoader {
	return &OrgLoader{pgLoaderBase: pgLoaderBase[seedinput.Org, models.Org]{
		params: params, fixture: f, name: ModelOrg,
	}}
}

func (l *OrgLoader) Defaults(ctx context.Context, input *seedinput.Org, fake bool) error {
	if input.CreatedAt.IsZero() {
		input.CreatedAt = l.fixture.clock.Now()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	if input.Budget == nil {
		budget := float64(models.DefaultBudget)
		input.Budget = &budget
	}

	if !fake {
		return nil
	}

	if input.Name == nil || *input.Name == "" {
		name := gofakeit.Company()
		input.Name = &name
	}
	if input.Status == nil || *input.Status == "" {
		statuses := []models.OrgStatus{
			models.OrgStatusActive,
			models.OrgStatusTrialing,
			models.OrgStatusTrialEnded,
			models.OrgStatusOverbudget,
			models.OrgStatusPaymentFailed,
		}
		// #nosec G404 -- math/rand is fine for fixtures
		status := statuses[rand.IntN(len(statuses))]
		input.Status = &status
	}
	return nil
}

func (l *OrgLoader) PopulateModel(ctx context.Context, model *models.Org, input *seedinput.Org) ([]string, error) {
	var cols []string
	if input.Name != nil && model.Name != *input.Name {
		model.Name = *input.Name
		cols = append(cols, "name")
	}
	if input.Status != nil && model.Status != *input.Status {
		model.Status = *input.Status
		cols = append(cols, "status")
	}
	if input.SubscriptionID != nil && model.SubscriptionID != *input.SubscriptionID {
		model.SubscriptionID = *input.SubscriptionID
		cols = append(cols, "subscription_id")
	}
	if input.OnPremises != nil && model.OnPremises != *input.OnPremises {
		model.OnPremises = *input.OnPremises
		cols = append(cols, "on_premises")
	}
	if input.Budget != nil && model.Budget != *input.Budget {
		model.Budget = *input.Budget
		cols = append(cols, "budget")
	}
	if !input.CreatedAt.IsZero() && model.CreatedAt != input.CreatedAt {
		model.CreatedAt = input.CreatedAt
		cols = append(cols, "created_at")
	}
	if !input.UpdatedAt.IsZero() && model.UpdatedAt != input.UpdatedAt {
		model.UpdatedAt = input.UpdatedAt
		cols = append(cols, "updated_at")
	}
	return cols, nil
}
