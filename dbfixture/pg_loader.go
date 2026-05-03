package dbfixture

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"

	"github.com/uptrace/uptrace/pkg/ctxkey"
	"github.com/uptrace/uptrace/pkg/json"
)

var bunTxKey = ctxkey.New[*bun.Tx]("dbfixture.bunTx", nil)

// IDer is implemented by models with a uint64 primary key.
type IDer interface {
	GetID() uint64
}

// pgLoaderBase is the base for all PostgreSQL fixture loaders.
// Concrete loaders embed this struct and override methods as needed.
//
// Fixture key → PK JSON mappings are stored centrally in Fixture.keys (KeyStore).
// Loaders access them via b.fixture.keys.Get/Set/Delete.
type pgLoaderBase[Input any, Model any] struct {
	params  LoaderParams
	fixture *Fixture
	name    string
	inputs  map[string]*Input
	loaded  map[string]*Entry[*Model]
}

// Entry holds a loaded model value and its insert status.
type Entry[T any] struct {
	Value    T
	Inserted bool
}

// ---------------------------------------------------------------------------
// LoadedStore implementation — typed getters
// ---------------------------------------------------------------------------

// Get returns a loaded model by key.
func (b *pgLoaderBase[Input, Model]) Get(key string) (*Model, bool) {
	if item, ok := b.loaded[key]; ok {
		return item.Value, true
	}
	return nil, false
}

// MustGet returns a loaded model or panics. Intended for tests.
func (b *pgLoaderBase[Input, Model]) MustGet(key string) *Model {
	v, ok := b.Get(key)
	if !ok {
		panic("dbfixture: " + b.name + "/" + key + " not found")
	}
	return v
}

// GetAny implements LoadedStore for type-erased access.
func (b *pgLoaderBase[Input, Model]) GetAny(key string) (any, bool) {
	if item, ok := b.loaded[key]; ok {
		return item.Value, true
	}
	return nil, false
}

// Loaded returns a snapshot map of all loaded entries (both inserted and
// pre-existing) keyed by fixture key. The returned map is a fresh copy so
// callers may iterate it after the seeding transaction has committed and
// while concurrent mutations to b.loaded occur (e.g., from event subscribers
// triggered during iteration).
func (b *pgLoaderBase[Input, Model]) Loaded() map[string]*Entry[*Model] {
	result := make(map[string]*Entry[*Model], len(b.loaded))
	for key, entry := range b.loaded {
		result[key] = entry
	}
	return result
}

// ---------------------------------------------------------------------------
// Infrastructure
// ---------------------------------------------------------------------------

// DB returns the database connection, using a transaction if one is active.
func (b *pgLoaderBase[Input, Model]) DB(ctx context.Context) bun.IDB {
	if tx, ok := bunTxKey.ValueOk(ctx); ok {
		return tx
	}
	return b.params.PG
}

// NewInput returns a zero-value input for YAML/JSON unmarshaling.
func (b *pgLoaderBase[Input, Model]) NewInput() *Input { return new(Input) }

// NewModel returns a zero-value model.
func (b *pgLoaderBase[Input, Model]) NewModel() *Model { return new(Model) }

// StoreInput stores the seedinput for later access by other loaders.
// If input is nil, the key is removed.
func (b *pgLoaderBase[Input, Model]) StoreInput(key string, input *Input) {
	if input == nil {
		delete(b.inputs, key)
		return
	}
	if b.inputs == nil {
		b.inputs = make(map[string]*Input)
	}
	b.inputs[key] = input
}

// GetInput returns the stored seedinput by key.
func (b *pgLoaderBase[Input, Model]) GetInput(key string) (*Input, bool) {
	if b.inputs == nil {
		return nil, false
	}
	v, ok := b.inputs[key]
	return v, ok
}

// StoreLoaded stores a model in the loaded map with the given insert status.
// If model is nil, the key is removed.
//
// KeyStore updates are the responsibility of the caller (Fixture.seedOne),
// which writes the fixture key → PK mapping after a successful Insert.
func (b *pgLoaderBase[Input, Model]) StoreLoaded(key string, model *Model, inserted bool) {
	if model == nil {
		delete(b.loaded, key)
		return
	}
	if b.loaded == nil {
		b.loaded = make(map[string]*Entry[*Model])
	}
	b.loaded[key] = &Entry[*Model]{Value: model, Inserted: inserted}
}

// ---------------------------------------------------------------------------
// Overridable methods — concrete loaders shadow these as needed
// ---------------------------------------------------------------------------

// Resolve resolves cross-references from already-loaded fixtures. No-op by default.
func (b *pgLoaderBase[Input, Model]) Resolve(ctx context.Context, input *Input) error {
	return nil
}

// Defaults applies default values to nil fields. When fake is true,
// remaining nil fields are populated with random data. No-op by default.
func (b *pgLoaderBase[Input, Model]) Defaults(ctx context.Context, input *Input, fake bool) error {
	return nil
}

// PopulateModel maps fields from input onto model and returns the column names
// that were set. No-op by default — concrete loaders MUST override this.
func (b *pgLoaderBase[Input, Model]) PopulateModel(ctx context.Context, model *Model, input *Input) ([]string, error) {
	return nil, nil
}

// Validate validates the model after it has been populated from input.
// No-op by default — concrete loaders override to call model.Validate()
// or other validation as appropriate.
func (b *pgLoaderBase[Input, Model]) Validate(ctx context.Context, model *Model) error {
	return nil
}

// ---------------------------------------------------------------------------
// Insert / Update — default implementations
// ---------------------------------------------------------------------------

// Insert inserts the model into the database. Pure DB operation — no side effects.
// Concrete loaders can shadow this method for custom insert logic.
func (b *pgLoaderBase[Input, Model]) Insert(ctx context.Context, model *Model) (bool, error) {
	if err := b.DB(ctx).NewInsert().Model(model).
		Returning("*").
		Scan(ctx, model); err != nil {
		return false, err
	}
	return true, nil
}

// Update patches the specified columns on the model. Pure DB operation — no side effects.
// Concrete loaders can shadow this method for custom update logic.
func (b *pgLoaderBase[Input, Model]) Update(ctx context.Context, model *Model, columns []string) error {
	return b.DB(ctx).NewUpdate().Model(model).
		Column(columns...).
		WherePK().
		Returning("*").
		Scan(ctx, model)
}

// ---------------------------------------------------------------------------
// PK helpers
// ---------------------------------------------------------------------------

// ModelPK returns the primary key columns for the model as a map of
// bun column name → value. The default implementation uses IDer.
// Composite-PK loaders must override this.
func (b *pgLoaderBase[Input, Model]) ModelPK(model *Model) map[string]any {
	if ider, ok := any(model).(IDer); ok {
		if id := ider.GetID(); id != 0 {
			return map[string]any{"id": id}
		}
	}
	return nil
}

// pkJSON returns the stored PK JSON for key, or an error if the fixture key
// is not registered. It is the single entry point for PK lookups.
func (b *pgLoaderBase[Input, Model]) pkJSON(key string) (json.RawMessage, error) {
	pkJSON, ok := b.fixture.keys.Get(b.name, key)
	if !ok {
		return nil, fmt.Errorf("dbfixture: %s/%s: fixture key not found (not loaded or deleted)", b.name, key)
	}
	return pkJSON, nil
}

// selectByPK reads the model from the database using the stored PK.
func (b *pgLoaderBase[Input, Model]) selectByPK(ctx context.Context, key string) (*Model, error) {
	pkJSON, err := b.pkJSON(key)
	if err != nil {
		return nil, err
	}
	var pk map[string]any
	if err := json.Unmarshal(pkJSON, &pk); err != nil {
		return nil, fmt.Errorf("dbfixture: %s/%s: unmarshal pk: %w", b.name, key, err)
	}
	model := new(Model)
	q := b.DB(ctx).NewSelect().Model(model)
	for col, val := range pk {
		q = q.Where("? = ?", bun.Ident(col), val)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return model, nil
}

// pkID extracts the uint64 "id" from the stored PK.
// Used by loaders with custom Select that need the raw ID
// (e.g. alert loaders that join additional tables).
func (b *pgLoaderBase[Input, Model]) pkID(key string) (uint64, bool) {
	pkJSON, err := b.pkJSON(key)
	if err != nil {
		return 0, false
	}
	var pk struct {
		ID uint64 `json:"id"`
	}
	if err := json.Unmarshal(pkJSON, &pk); err != nil {
		return 0, false
	}
	return pk.ID, pk.ID != 0
}

// ---------------------------------------------------------------------------
// Non-overridable methods
// ---------------------------------------------------------------------------

// Delete removes the record identified by the fixture key.
// If the model is already in the loaded map it is used directly;
// otherwise the record is fetched from the database by its persisted PK.
func (b *pgLoaderBase[Input, Model]) Delete(ctx context.Context, key string) error {
	var model *Model
	if entry, ok := b.loaded[key]; ok {
		model = entry.Value
	} else {
		var err error
		model, err = b.selectByPK(ctx, key)
		if err != nil {
			return err
		}
	}

	if _, err := b.DB(ctx).NewDelete().Model(model).WherePK().Exec(ctx); err != nil {
		return err
	}

	if err := b.fixture.keys.Delete(b.name, key); err != nil {
		return err
	}
	delete(b.loaded, key)
	return nil
}

// Select reads the record back from the database by its fixture key
// and stores it in the loaded map.
func (b *pgLoaderBase[Input, Model]) Select(ctx context.Context, key string) (*Model, error) {
	model, err := b.selectByPK(ctx, key)
	if err != nil {
		return nil, err
	}
	b.StoreLoaded(key, model, false)
	return model, nil
}

