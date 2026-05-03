package dbfixture

import (
	"context"
	"fmt"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// ---------------------------------------------------------------------------
// DashboardLoader
// ---------------------------------------------------------------------------

// DashboardLoader loads dashboard fixtures.
type DashboardLoader struct {
	pgLoaderBase[seedinput.Dashboard, models.Dashboard]
}

// NewDashboardLoader creates a Loader for dashboards.
func NewDashboardLoader(f *Fixture, params LoaderParams) *DashboardLoader {
	return &DashboardLoader{pgLoaderBase: pgLoaderBase[seedinput.Dashboard, models.Dashboard]{
		params: params, fixture: f, name: ModelDashboard,
	}}
}

func (l *DashboardLoader) Resolve(ctx context.Context, input *seedinput.Dashboard) error {
	return resolveProjectID(l.fixture, &input.ProjectID, input.ProjectKey)
}

func (l *DashboardLoader) Defaults(ctx context.Context, input *seedinput.Dashboard, fake bool) error {
	if input.CreatedAt.IsZero() {
		input.CreatedAt = l.fixture.clock.Now()
	}
	if input.UpdatedAt.IsZero() {
		input.UpdatedAt = input.CreatedAt
	}
	return nil
}

func (l *DashboardLoader) PopulateModel(ctx context.Context, model *models.Dashboard, input *seedinput.Dashboard) ([]string, error) {
	var cols []string
	if input.ProjectID != 0 && model.ProjectID != input.ProjectID {
		model.ProjectID = input.ProjectID
		cols = append(cols, "project_id")
	}
	if input.Name != nil && model.Name != *input.Name {
		model.Name = *input.Name
		cols = append(cols, "name")
	}
	if input.Pinned && model.Pinned != input.Pinned {
		model.Pinned = input.Pinned
		cols = append(cols, "pinned")
	}
	if input.SortingOrder != 0 && model.SortingOrder != input.SortingOrder {
		model.SortingOrder = input.SortingOrder
		cols = append(cols, "sorting_order")
	}
	if input.Tags != nil {
		model.Tags = input.Tags
		cols = append(cols, "tags")
	}
	if input.Table != nil {
		model.Table = *input.Table
		cols = append(cols, "table")
	}
	if input.GridVariables != nil {
		model.GridVariables = input.GridVariables
		cols = append(cols, "grid_variables")
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
// ProjectTagLoader
// ---------------------------------------------------------------------------

// ProjectTagLoader loads project tag fixtures.
type ProjectTagLoader struct {
	pgLoaderBase[seedinput.ProjectTag, models.ProjectTag]
}

// NewProjectTagLoader creates a Loader for project tags.
func NewProjectTagLoader(f *Fixture, params LoaderParams) *ProjectTagLoader {
	return &ProjectTagLoader{pgLoaderBase: pgLoaderBase[seedinput.ProjectTag, models.ProjectTag]{
		params: params, fixture: f, name: ModelProjectTag,
	}}
}

func (l *ProjectTagLoader) Resolve(ctx context.Context, input *seedinput.ProjectTag) error {
	return resolveProjectID(l.fixture, &input.ProjectID, input.ProjectKey)
}

func (l *ProjectTagLoader) PopulateModel(ctx context.Context, model *models.ProjectTag, input *seedinput.ProjectTag) ([]string, error) {
	var cols []string
	if input.ProjectID != 0 {
		model.ProjectID = input.ProjectID
		cols = append(cols, "project_id")
	}
	if input.Name != "" {
		model.Name = input.Name
		cols = append(cols, "name")
	}
	if input.SortingOrder != 0 {
		model.SortingOrder = input.SortingOrder
		cols = append(cols, "sorting_order")
	}
	return cols, nil
}

func (l *ProjectTagLoader) Insert(ctx context.Context, model *models.ProjectTag) (bool, error) {
	db := l.DB(ctx)
	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	// Re-select by natural key.
	if err := db.NewSelect().Model(model).
		Where("project_id = ?", model.ProjectID).
		Where("name = ?", model.Name).
		Scan(ctx); err != nil {
		return false, err
	}
	return n != 0, nil
}

// ---------------------------------------------------------------------------
// TaggedDashboardLoader
// ---------------------------------------------------------------------------

// TaggedDashboardLoader loads tagged dashboard relationship fixtures.
type TaggedDashboardLoader struct {
	pgLoaderBase[seedinput.TaggedDashboard, models.TaggedDashboard]
}

// NewTaggedDashboardLoader creates a Loader for tagged dashboard relationships.
func NewTaggedDashboardLoader(f *Fixture, params LoaderParams) *TaggedDashboardLoader {
	return &TaggedDashboardLoader{pgLoaderBase: pgLoaderBase[seedinput.TaggedDashboard, models.TaggedDashboard]{
		params: params, fixture: f, name: ModelTaggedDashboard,
	}}
}

func (l *TaggedDashboardLoader) Resolve(ctx context.Context, input *seedinput.TaggedDashboard) error {
	tag, ok := Get[*models.ProjectTag](l.fixture, input.TagKey)
	if !ok {
		return fmt.Errorf("project_tag %q not found", input.TagKey)
	}
	input.TagID = tag.ID

	dashboard, ok := Get[*models.Dashboard](l.fixture, input.DashboardKey)
	if !ok {
		return fmt.Errorf("dashboard %q not found", input.DashboardKey)
	}
	input.DashboardID = dashboard.ID
	return nil
}

func (l *TaggedDashboardLoader) PopulateModel(ctx context.Context, model *models.TaggedDashboard, input *seedinput.TaggedDashboard) ([]string, error) {
	model.TagID = input.TagID
	model.DashboardID = input.DashboardID
	model.SortingOrder = input.SortingOrder
	return []string{"tag_id", "dashboard_id", "sorting_order"}, nil
}

func (l *TaggedDashboardLoader) Insert(ctx context.Context, model *models.TaggedDashboard) (bool, error) {
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
		Where("tag_id = ?", model.TagID).
		Where("dashboard_id = ?", model.DashboardID).
		Scan(ctx); err != nil {
		return false, err
	}
	return n != 0, nil
}

func (l *TaggedDashboardLoader) ModelPK(model *models.TaggedDashboard) map[string]any {
	return map[string]any{"tag_id": model.TagID, "dashboard_id": model.DashboardID}
}

func (l *TaggedDashboardLoader) Update(ctx context.Context, model *models.TaggedDashboard, columns []string) error {
	return nil // join table — nothing to update besides sorting_order which is always set via Insert
}
