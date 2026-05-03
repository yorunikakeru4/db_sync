package dbfixture

import (
	"context"
	"fmt"

	"github.com/brianvoe/gofakeit/v5"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// ---------------------------------------------------------------------------
// ProjectLoader
// ---------------------------------------------------------------------------

// ProjectLoader loads project fixtures.
type ProjectLoader struct {
	pgLoaderBase[seedinput.Project, models.Project]
}

// NewProjectLoader creates a Loader for projects.
func NewProjectLoader(f *Fixture, params LoaderParams) *ProjectLoader {
	return &ProjectLoader{pgLoaderBase: pgLoaderBase[seedinput.Project, models.Project]{
		params: params, fixture: f, name: ModelProject,
	}}
}

func (l *ProjectLoader) Resolve(ctx context.Context, input *seedinput.Project) error {
	if input.OrgID != 0 {
		return nil
	}
	if p := l.fixture.overrideProject(); p != nil {
		input.OrgID = p.OrgID
		return nil
	}
	if input.OrgKey == "" {
		// No org specified — caller is updating an existing project
		// and wants to keep the current org_id.
		return nil
	}
	org, ok := Get[*models.Org](l.fixture, input.OrgKey)
	if !ok {
		return fmt.Errorf("org %q not found", input.OrgKey)
	}
	input.OrgID = org.ID
	return nil
}

func (l *ProjectLoader) Defaults(ctx context.Context, input *seedinput.Project, fake bool) error {
	if input.CreatedAt.IsZero() {
		input.CreatedAt = l.fixture.clock.Now()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	if input.SpanColumns == nil || len(*input.SpanColumns) == 0 {
		input.SpanColumns = &models.DefaultSpanColumns
	}
	if input.LogColumns == nil || len(*input.LogColumns) == 0 {
		input.LogColumns = &models.DefaultLogColumns
	}
	if input.EventColumns == nil || len(*input.EventColumns) == 0 {
		input.EventColumns = &models.DefaultEventColumns
	}
	if input.TraceColumns == nil || len(*input.TraceColumns) == 0 {
		input.TraceColumns = &models.DefaultTraceColumns
	}
	if input.SpanQuery == nil || *input.SpanQuery == "" {
		input.SpanQuery = &models.DefaultSpanQuery
	}
	if input.LogQuery == nil || *input.LogQuery == "" {
		input.LogQuery = &models.DefaultLogQuery
	}
	if input.EventQuery == nil || *input.EventQuery == "" {
		input.EventQuery = &models.DefaultEventQuery
	}
	if input.TraceQuery == nil || *input.TraceQuery == "" {
		input.TraceQuery = &models.DefaultTraceQuery
	}

	if fake {
		if input.Name == nil || *input.Name == "" {
			name := gofakeit.AppName()
			input.Name = &name
		}
	}
	return nil
}

func (l *ProjectLoader) PopulateModel(ctx context.Context, model *models.Project, input *seedinput.Project) ([]string, error) {
	var cols []string
	if input.OrgID != 0 && model.OrgID != input.OrgID {
		model.OrgID = input.OrgID
		cols = append(cols, "org_id")
	}
	if input.Name != nil && model.Name != *input.Name {
		model.Name = *input.Name
		cols = append(cols, "name")
	}
	if input.Suspended != nil && model.Suspended != *input.Suspended {
		model.Suspended = *input.Suspended
		cols = append(cols, "suspended")
	}
	if input.SpanQuery != nil && model.SpanQuery != *input.SpanQuery {
		model.SpanQuery = *input.SpanQuery
		cols = append(cols, "span_query")
	}
	if input.SpanColumns != nil {
		model.SpanColumns = *input.SpanColumns
		cols = append(cols, "span_columns")
	}
	if input.SpanRetention != nil && model.SpanRetention != *input.SpanRetention {
		model.SpanRetention = *input.SpanRetention
		cols = append(cols, "span_retention")
	}
	if input.LogQuery != nil && model.LogQuery != *input.LogQuery {
		model.LogQuery = *input.LogQuery
		cols = append(cols, "log_query")
	}
	if input.LogColumns != nil {
		model.LogColumns = *input.LogColumns
		cols = append(cols, "log_columns")
	}
	if input.LogRetention != nil && model.LogRetention != *input.LogRetention {
		model.LogRetention = *input.LogRetention
		cols = append(cols, "log_retention")
	}
	if input.EventQuery != nil && model.EventQuery != *input.EventQuery {
		model.EventQuery = *input.EventQuery
		cols = append(cols, "event_query")
	}
	if input.EventColumns != nil {
		model.EventColumns = *input.EventColumns
		cols = append(cols, "event_columns")
	}
	if input.EventRetention != nil && model.EventRetention != *input.EventRetention {
		model.EventRetention = *input.EventRetention
		cols = append(cols, "event_retention")
	}
	if input.TraceQuery != nil && model.TraceQuery != *input.TraceQuery {
		model.TraceQuery = *input.TraceQuery
		cols = append(cols, "trace_query")
	}
	if input.TraceColumns != nil {
		model.TraceColumns = *input.TraceColumns
		cols = append(cols, "trace_columns")
	}
	if input.MetricRetention != nil && model.MetricRetention != *input.MetricRetention {
		model.MetricRetention = *input.MetricRetention
		cols = append(cols, "metric_retention")
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

// ---------------------------------------------------------------------------
// ProjectTokenLoader
// ---------------------------------------------------------------------------

// ProjectTokenLoader loads project token fixtures.
type ProjectTokenLoader struct {
	pgLoaderBase[seedinput.ProjectToken, models.ProjectToken]
}

// NewProjectTokenLoader creates a Loader for project tokens.
func NewProjectTokenLoader(f *Fixture, params LoaderParams) *ProjectTokenLoader {
	return &ProjectTokenLoader{pgLoaderBase: pgLoaderBase[seedinput.ProjectToken, models.ProjectToken]{
		params: params, fixture: f, name: ModelProjectToken,
	}}
}

func (l *ProjectTokenLoader) Resolve(ctx context.Context, input *seedinput.ProjectToken) error {
	return resolveProjectID(l.fixture, &input.ProjectID, input.ProjectKey)
}

func (l *ProjectTokenLoader) Defaults(ctx context.Context, input *seedinput.ProjectToken, fake bool) error {
	if fake && input.Token == "" {
		input.Token = gofakeit.UUID()
	}
	return nil
}

func (l *ProjectTokenLoader) PopulateModel(ctx context.Context, model *models.ProjectToken, input *seedinput.ProjectToken) ([]string, error) {
	var cols []string
	if input.ProjectID != 0 && model.ProjectID != input.ProjectID {
		model.ProjectID = input.ProjectID
		cols = append(cols, "project_id")
	}
	if input.Token != "" && model.Token != input.Token {
		model.Token = input.Token
		cols = append(cols, "token")
	}
	if !input.CreatedAt.IsZero() && model.CreatedAt != input.CreatedAt {
		model.CreatedAt = input.CreatedAt
		cols = append(cols, "created_at")
	}
	return cols, nil
}

// ---------------------------------------------------------------------------
// ProjectUserLoader
// ---------------------------------------------------------------------------

// ProjectUserLoader loads project-user relationship fixtures.
type ProjectUserLoader struct {
	pgLoaderBase[seedinput.ProjectUser, models.OrgUserProject]
}

// NewProjectUserLoader creates a Loader for project-user relationships.
func NewProjectUserLoader(f *Fixture, params LoaderParams) *ProjectUserLoader {
	return &ProjectUserLoader{pgLoaderBase: pgLoaderBase[seedinput.ProjectUser, models.OrgUserProject]{
		params: params, fixture: f, name: ModelProjectUser,
	}}
}

func (l *ProjectUserLoader) Resolve(ctx context.Context, input *seedinput.ProjectUser) error {
	project, ok := Get[*models.Project](l.fixture, input.ProjectKey)
	if !ok {
		return fmt.Errorf("project %q not found", input.ProjectKey)
	}
	input.ProjectID = project.ID
	input.OrgID = project.OrgID

	orgUser, ok := Get[*models.OrgUser](l.fixture, input.OrgUserKey)
	if !ok {
		return fmt.Errorf("org_user %q not found", input.OrgUserKey)
	}
	input.OrgUserID = orgUser.ID
	return nil
}

func (l *ProjectUserLoader) PopulateModel(ctx context.Context, model *models.OrgUserProject, input *seedinput.ProjectUser) ([]string, error) {
	var cols []string
	if input.OrgID != 0 && model.OrgID != input.OrgID {
		model.OrgID = input.OrgID
		cols = append(cols, "org_id")
	}
	if input.ProjectID != 0 && model.ProjectID != input.ProjectID {
		model.ProjectID = input.ProjectID
		cols = append(cols, "project_id")
	}
	if input.OrgUserID != 0 && model.OrgUserID != input.OrgUserID {
		model.OrgUserID = input.OrgUserID
		cols = append(cols, "org_user_id")
	}
	if input.PermLevel != nil && model.PermLevel != *input.PermLevel {
		model.PermLevel = *input.PermLevel
		cols = append(cols, "perm_level")
	}
	return cols, nil
}

func (l *ProjectUserLoader) Insert(ctx context.Context, model *models.OrgUserProject) (bool, error) {
	db := l.DB(ctx)
	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()

	if err := db.NewSelect().Model(model).
		Where("org_user_id = ?", model.OrgUserID).
		Where("project_id = ?", model.ProjectID).
		Scan(ctx); err != nil {
		return false, err
	}
	return n != 0, nil
}

func (l *ProjectUserLoader) ModelPK(model *models.OrgUserProject) map[string]any {
	return map[string]any{"org_user_id": model.OrgUserID, "project_id": model.ProjectID}
}

func (l *ProjectUserLoader) Update(ctx context.Context, model *models.OrgUserProject, columns []string) error {
	return nil // join table — nothing to update
}
