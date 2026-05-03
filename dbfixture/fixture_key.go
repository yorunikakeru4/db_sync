package dbfixture

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/uptrace/uptrace/pkg/json"
)

// FixtureKey maps a (model, key) pair to the primary key of a seeded row.
// This replaces the old _ns/_key columns that were embedded in every table.
type FixtureKey struct {
	bun.BaseModel `bun:"fixture_keys,alias:fk"`

	Model string          `bun:",pk"`
	Key   string          `bun:",pk"`
	PK    json.RawMessage `bun:"pk,type:jsonb"`
}

// FixtureKeyRepo provides CRUD operations on the fixture_keys table.
type FixtureKeyRepo struct {
	db *bun.DB
}

// NewFixtureKeyRepo creates a new FixtureKeyRepo.
func NewFixtureKeyRepo(db *bun.DB) *FixtureKeyRepo {
	return &FixtureKeyRepo{db: db}
}

// DB returns the database connection, using a transaction if one is active.
func (r *FixtureKeyRepo) DB(ctx context.Context) bun.IDB {
	if tx, ok := bunTxKey.ValueOk(ctx); ok {
		return tx
	}
	return r.db
}

// SaveKey upserts a fixture key mapping.
func (r *FixtureKeyRepo) SaveKey(ctx context.Context, model, key string, pk any) error {
	pkJSON, err := json.Marshal(pk)
	if err != nil {
		return err
	}
	fk := &FixtureKey{
		Model: model,
		Key:   key,
		PK:    pkJSON,
	}
	_, err = r.DB(ctx).NewInsert().Model(fk).
		On("CONFLICT (model, key) DO UPDATE").
		Set("pk = EXCLUDED.pk").
		Exec(ctx)
	return err
}

// LoadKey reads the PK for a given (model, key) pair.
func (r *FixtureKeyRepo) LoadKey(ctx context.Context, model, key string) (json.RawMessage, error) {
	fk := new(FixtureKey)
	if err := r.DB(ctx).NewSelect().Model(fk).
		Where("model = ?", model).
		Where("key = ?", key).
		Scan(ctx); err != nil {
		return nil, err
	}
	return fk.PK, nil
}

// DeleteKey removes a fixture key mapping.
func (r *FixtureKeyRepo) DeleteKey(ctx context.Context, model, key string) error {
	_, err := r.DB(ctx).NewDelete().Model((*FixtureKey)(nil)).
		Where("model = ?", model).
		Where("key = ?", key).
		Exec(ctx)
	return err
}

// ListKeys returns all fixture keys for a given model.
func (r *FixtureKeyRepo) ListKeys(ctx context.Context, model string) ([]FixtureKey, error) {
	var keys []FixtureKey
	if err := r.DB(ctx).NewSelect().Model(&keys).
		Where("model = ?", model).
		Scan(ctx); err != nil {
		return nil, err
	}
	return keys, nil
}

// ListAllKeys returns all fixture keys across all models.
func (r *FixtureKeyRepo) ListAllKeys(ctx context.Context) ([]FixtureKey, error) {
	var keys []FixtureKey
	if err := r.DB(ctx).NewSelect().Model(&keys).Scan(ctx); err != nil {
		return nil, err
	}
	return keys, nil
}

// SaveKeys batch-upserts multiple fixture key mappings.
func (r *FixtureKeyRepo) SaveKeys(ctx context.Context, keys []FixtureKey) error {
	if len(keys) == 0 {
		return nil
	}
	_, err := r.DB(ctx).NewInsert().Model(&keys).
		On("CONFLICT (model, key) DO UPDATE").
		Set("pk = EXCLUDED.pk").
		Exec(ctx)
	return err
}

// DeleteAll removes all fixture keys.
func (r *FixtureKeyRepo) DeleteAll(ctx context.Context) error {
	_, err := r.DB(ctx).NewDelete().Model((*FixtureKey)(nil)).WhereAllWithDeleted().Exec(ctx)
	return err
}

// DeleteByModel removes all fixture keys for a given model.
func (r *FixtureKeyRepo) DeleteByModel(ctx context.Context, model string) error {
	_, err := r.DB(ctx).NewDelete().Model((*FixtureKey)(nil)).
		Where("model = ?", model).
		Exec(ctx)
	return err
}

// DeleteKeys removes fixture keys for a given model with the specified keys.
func (r *FixtureKeyRepo) DeleteKeys(ctx context.Context, model string, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	_, err := r.DB(ctx).NewDelete().Model((*FixtureKey)(nil)).
		Where("model = ?", model).
		Where("key IN ?", bun.Tuple(keys)).
		Exec(ctx)
	return err
}
