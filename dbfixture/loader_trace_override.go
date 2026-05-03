package dbfixture

import (
	"context"
	"fmt"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// TraceOverrideLoader loads trace override fixtures.
// TraceOverride uses project_id + trace_id as its PK, not _key/_ns.
type TraceOverrideLoader struct {
	pgLoaderBase[seedinput.TraceOverride, models.TraceOverride]
}

// NewTraceOverrideLoader creates a Loader for trace overrides.
func NewTraceOverrideLoader(f *Fixture, params LoaderParams) *TraceOverrideLoader {
	return &TraceOverrideLoader{pgLoaderBase: pgLoaderBase[seedinput.TraceOverride, models.TraceOverride]{
		params: params, fixture: f, name: ModelTraceOverride,
	}}
}

func (l *TraceOverrideLoader) Resolve(ctx context.Context, input *seedinput.TraceOverride) error {
	if input.UserKey != "" {
		user, ok := Get[*models.User](l.fixture, input.UserKey)
		if !ok {
			return fmt.Errorf("user %q not found", input.UserKey)
		}
		input.UserID = user.ID
	}
	return nil
}

func (l *TraceOverrideLoader) PopulateModel(ctx context.Context, model *models.TraceOverride, input *seedinput.TraceOverride) ([]string, error) {
	project, ok := Get[*models.Project](l.fixture, input.ProjectKey)
	if !ok {
		return nil, fmt.Errorf("project %q not found", input.ProjectKey)
	}
	model.ProjectID = project.ID
	model.TraceID = input.TraceID
	model.ExpiresAt = input.ExpiresAt
	model.UserID = input.UserID
	if input.Reason != nil {
		model.Reason = *input.Reason
	}
	return nil, nil // columns not used — custom Insert with upsert
}

func (l *TraceOverrideLoader) Validate(ctx context.Context, model *models.TraceOverride) error {
	return model.Validate()
}

func (l *TraceOverrideLoader) Insert(ctx context.Context, model *models.TraceOverride) (bool, error) {
	db := l.DB(ctx)
	if err := db.NewInsert().
		Model(model).
		On("CONFLICT (project_id, trace_id) DO UPDATE").
		Set("expires_at = EXCLUDED.expires_at").
		Set("user_id = EXCLUDED.user_id").
		Set("reason = EXCLUDED.reason").
		Returning("*").
		Scan(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (l *TraceOverrideLoader) ModelPK(model *models.TraceOverride) map[string]any {
	if model.ID != 0 {
		return map[string]any{"id": model.ID}
	}
	return nil
}
