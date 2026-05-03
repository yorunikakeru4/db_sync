package dbfixture

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/brianvoe/gofakeit/v5"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// ---------------------------------------------------------------------------
// TeamLoader
// ---------------------------------------------------------------------------

// TeamLoader loads team fixtures.
type TeamLoader struct {
	pgLoaderBase[seedinput.Team, models.Team]
}

// NewTeamLoader creates a Loader for teams.
func NewTeamLoader(f *Fixture, params LoaderParams) *TeamLoader {
	return &TeamLoader{pgLoaderBase: pgLoaderBase[seedinput.Team, models.Team]{
		params: params, fixture: f, name: ModelTeam,
	}}
}

func (l *TeamLoader) Resolve(ctx context.Context, input *seedinput.Team) error {
	org, ok := Get[*models.Org](l.fixture, input.OrgKey)
	if !ok {
		return fmt.Errorf("org %q not found", input.OrgKey)
	}
	input.OrgID = org.ID
	return nil
}

func (l *TeamLoader) Defaults(ctx context.Context, input *seedinput.Team, fake bool) error {
	if input.CreatedAt.IsZero() {
		input.CreatedAt = l.fixture.clock.Now()
	}

	if !fake {
		return nil
	}

	if input.Name == nil || *input.Name == "" {
		name := gofakeit.Name()
		input.Name = &name
	}
	if input.PermLevel == nil || *input.PermLevel == "" {
		permLevels := []models.PermLevel{
			models.PermLevelNone,
			models.PermLevelView,
			models.PermLevelEdit,
			models.PermLevelAdmin,
		}
		// #nosec G404 -- math/rand is fine for fixtures
		perm := permLevels[rand.IntN(len(permLevels))]
		input.PermLevel = &perm
	}
	return nil
}

func (l *TeamLoader) PopulateModel(ctx context.Context, model *models.Team, input *seedinput.Team) ([]string, error) {
	var cols []string
	if input.OrgID != 0 && model.OrgID != input.OrgID {
		model.OrgID = input.OrgID
		cols = append(cols, "org_id")
	}
	if input.Name != nil && model.Name != *input.Name {
		model.Name = *input.Name
		cols = append(cols, "name")
	}
	if input.PermLevel != nil && model.PermLevel != *input.PermLevel {
		model.PermLevel = *input.PermLevel
		cols = append(cols, "perm_level")
	}
	if !input.CreatedAt.IsZero() && model.CreatedAt != input.CreatedAt {
		model.CreatedAt = input.CreatedAt
		cols = append(cols, "created_at")
	}
	return cols, nil
}

// ---------------------------------------------------------------------------
// TeamUserLoader
// ---------------------------------------------------------------------------

// TeamUserLoader loads team-user relationship fixtures.
type TeamUserLoader struct {
	pgLoaderBase[seedinput.TeamUser, models.TeamUser]
}

// NewTeamUserLoader creates a Loader for team-user relationships.
func NewTeamUserLoader(f *Fixture, params LoaderParams) *TeamUserLoader {
	return &TeamUserLoader{pgLoaderBase: pgLoaderBase[seedinput.TeamUser, models.TeamUser]{
		params: params, fixture: f, name: ModelTeamUser,
	}}
}

func (l *TeamUserLoader) Resolve(ctx context.Context, input *seedinput.TeamUser) error {
	team, ok := Get[*models.Team](l.fixture, input.TeamKey)
	if !ok {
		return fmt.Errorf("team %q not found", input.TeamKey)
	}
	input.TeamID = team.ID

	orgUser, ok := Get[*models.OrgUser](l.fixture, input.OrgUserKey)
	if !ok {
		return fmt.Errorf("org_user %q not found", input.OrgUserKey)
	}
	input.OrgUserID = orgUser.ID
	return nil
}

func (l *TeamUserLoader) PopulateModel(ctx context.Context, model *models.TeamUser, input *seedinput.TeamUser) ([]string, error) {
	model.TeamID = input.TeamID
	model.OrgUserID = input.OrgUserID
	return []string{"team_id", "org_user_id"}, nil
}

func (l *TeamUserLoader) Insert(ctx context.Context, model *models.TeamUser) (bool, error) {
	db := l.DB(ctx)
	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if err := db.NewSelect().Model(model).
		Where("team_id = ?", model.TeamID).
		Where("org_user_id = ?", model.OrgUserID).
		Scan(ctx); err != nil {
		return false, err
	}
	return n != 0, nil
}

func (l *TeamUserLoader) ModelPK(model *models.TeamUser) map[string]any {
	return map[string]any{"team_id": model.TeamID, "org_user_id": model.OrgUserID}
}

func (l *TeamUserLoader) Update(ctx context.Context, model *models.TeamUser, columns []string) error {
	return nil // all columns are PK — nothing to update
}

// ---------------------------------------------------------------------------
// TeamProjectLoader
// ---------------------------------------------------------------------------

// TeamProjectLoader loads team-project relationship fixtures.
type TeamProjectLoader struct {
	pgLoaderBase[seedinput.TeamProject, models.TeamProject]
}

// NewTeamProjectLoader creates a Loader for team-project relationships.
func NewTeamProjectLoader(f *Fixture, params LoaderParams) *TeamProjectLoader {
	return &TeamProjectLoader{pgLoaderBase: pgLoaderBase[seedinput.TeamProject, models.TeamProject]{
		params: params, fixture: f, name: ModelTeamProject,
	}}
}

func (l *TeamProjectLoader) Resolve(ctx context.Context, input *seedinput.TeamProject) error {
	team, ok := Get[*models.Team](l.fixture, input.TeamKey)
	if !ok {
		return fmt.Errorf("team %q not found", input.TeamKey)
	}
	input.TeamID = team.ID

	project, ok := Get[*models.Project](l.fixture, input.ProjectKey)
	if !ok {
		return fmt.Errorf("project %q not found", input.ProjectKey)
	}
	input.ProjectID = project.ID
	return nil
}

func (l *TeamProjectLoader) PopulateModel(ctx context.Context, model *models.TeamProject, input *seedinput.TeamProject) ([]string, error) {
	model.TeamID = input.TeamID
	model.ProjectID = input.ProjectID
	return []string{"team_id", "project_id"}, nil
}

func (l *TeamProjectLoader) Insert(ctx context.Context, model *models.TeamProject) (bool, error) {
	db := l.DB(ctx)
	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	if err := db.NewSelect().Model(model).
		Where("team_id = ?", model.TeamID).
		Where("project_id = ?", model.ProjectID).
		Scan(ctx); err != nil {
		return false, err
	}
	return n != 0, nil
}

func (l *TeamProjectLoader) ModelPK(model *models.TeamProject) map[string]any {
	return map[string]any{"team_id": model.TeamID, "project_id": model.ProjectID}
}

func (l *TeamProjectLoader) Update(ctx context.Context, model *models.TeamProject, columns []string) error {
	return nil // all columns are PK — nothing to update
}
