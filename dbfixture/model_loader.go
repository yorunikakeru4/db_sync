package dbfixture

import (
	"context"
	"fmt"
	"reflect"

	"github.com/uptrace/bun"

	"github.com/uptrace/uptrace/bootstrap"
)

// Loader[Input, Model] handles the full lifecycle for a fixture type.
//
// Input is the seedinput type used for YAML/JSON unmarshaling. Pointer fields in Input
// distinguish "explicitly set" from "not provided", which is critical for partial updates.
//
// Model is the DB model type. Each loader stores loaded models internally
// and provides typed getters.
type Loader[Input any, Model any] interface {
	// --- Construction ---

	// NewInput returns a zero-value input for YAML/JSON unmarshaling.
	NewInput() Input

	// NewModel returns a zero-value model for populating from input.
	NewModel() Model

	// --- Input preparation (run in order before PopulateModel) ---

	// Resolve resolves cross-references from already-loaded fixtures.
	Resolve(ctx context.Context, input Input) error

	// Defaults fills nil pointer fields with sensible default values.
	// When fake is true, remaining nil fields are populated with random
	// data via gofakeit.
	Defaults(ctx context.Context, input Input, fake bool) error

	// --- Input → model ---

	// PopulateModel maps fields from input onto model and returns the list
	// of DB column names that were set. This is a pure function — no side effects.
	PopulateModel(ctx context.Context, model Model, input Input) ([]string, error)

	// Validate validates the model after it has been populated from input.
	Validate(ctx context.Context, model Model) error

	// --- Database operations ---

	// Insert inserts the model into the database. Returns inserted=true if a new
	// row was created. This is a pure DB operation — no side effects.
	Insert(ctx context.Context, model Model) (inserted bool, err error)

	// Update patches the specified columns on the model. The model must already
	// have its ID from a prior Select. This is a pure DB operation — no side effects.
	Update(ctx context.Context, model Model, columns []string) error

	// Select reads the record back from the database by its fixture key.
	Select(ctx context.Context, key string) (Model, error)

	// Delete removes the record identified by the fixture key.
	Delete(ctx context.Context, key string) error

	// ModelPK returns the PK column→value map for a model. The map is stored in
	// the central KeyStore so the row can be located later.
	// Default uses IDer; composite-PK loaders must override.
	ModelPK(model Model) map[string]any

	// --- In-memory bookkeeping (called after Insert/Update) ---

	// StoreInput stores the input for later access by other loaders.
	StoreInput(key string, input Input)

	// StoreLoaded stores the model in the loaded map with the given insert status.
	StoreLoaded(key string, model Model, inserted bool)
}

// ---------------------------------------------------------------------------
// AnyLoader — type-erased interface for the registry
// ---------------------------------------------------------------------------

// AnyLoader is the type-erased version of Loader stored in the registry.
type AnyLoader interface {
	NewInput() any
	NewModel() any

	// Per-item operations — used by ddtest crud_command directly.
	Resolve(ctx context.Context, input any) error
	Defaults(ctx context.Context, input any, fake bool) error
	PopulateModel(ctx context.Context, model any, input any) ([]string, error)
	Validate(ctx context.Context, model any) error
	Insert(ctx context.Context, model any) (inserted bool, err error)
	Update(ctx context.Context, model any, columns []string) error
	StoreLoaded(key string, model any, inserted bool)
	StoreInput(key string, input any)
	Delete(ctx context.Context, key string) error
	Select(ctx context.Context, key string) (any, error)

	// ModelPK returns the PK column→value map for a model.
	ModelPK(model any) map[string]any

	// Loaded model access — type-erased, string keys.
	Get(key string) (any, bool)
}

// ---------------------------------------------------------------------------
// OptApplier — optional interface for loaders that accept per-loader options
// ---------------------------------------------------------------------------

// AfterClearer is optionally implemented by loaders that need cleanup
// after all loaded records have been deleted (e.g. truncating
// pre-aggregated tables in ClickHouse).
type AfterClearer interface {
	AfterClear(ctx context.Context) error
}

// OptApplier is optionally implemented by loaders that accept per-loader options.
// Loaders receive options during Fixture.Register via type assertion.
type OptApplier interface {
	ApplyOpt(opt any)
}

type modelTyper interface {
	ModelType() reflect.Type
}

// ---------------------------------------------------------------------------
// Wrap — adapts a typed Loader into an AnyLoader
// ---------------------------------------------------------------------------

// Wrap adapts a typed Loader[Input, Model] into an AnyLoader.
func Wrap[Input any, Model any](l Loader[Input, Model]) AnyLoader {
	return &loaderAdapter[Input, Model]{inner: l}
}

type loaderAdapter[Input any, Model any] struct {
	inner Loader[Input, Model]
}

func (a *loaderAdapter[Input, Model]) ModelType() reflect.Type {
	return reflect.TypeFor[Model]()
}

// Keyer is implemented by fixture input types to return their fixture key.
type Keyer interface {
	FixtureKey() string
}

func (a *loaderAdapter[Input, Model]) NewInput() any {
	return a.inner.NewInput()
}
func (a *loaderAdapter[Input, Model]) NewModel() any {
	return a.inner.NewModel()
}
func (a *loaderAdapter[Input, Model]) Resolve(ctx context.Context, input any) error {
	return a.inner.Resolve(ctx, input.(Input))
}
func (a *loaderAdapter[Input, Model]) Defaults(ctx context.Context, input any, fake bool) error {
	return a.inner.Defaults(ctx, input.(Input), fake)
}
func (a *loaderAdapter[Input, Model]) PopulateModel(ctx context.Context, model any, input any) ([]string, error) {
	return a.inner.PopulateModel(ctx, model.(Model), input.(Input))
}
func (a *loaderAdapter[Input, Model]) Validate(ctx context.Context, model any) error {
	return a.inner.Validate(ctx, model.(Model))
}
func (a *loaderAdapter[Input, Model]) Insert(ctx context.Context, model any) (bool, error) {
	return a.inner.Insert(ctx, model.(Model))
}
func (a *loaderAdapter[Input, Model]) Update(ctx context.Context, model any, columns []string) error {
	return a.inner.Update(ctx, model.(Model), columns)
}
func (a *loaderAdapter[Input, Model]) StoreLoaded(key string, model any, inserted bool) {
	if model == nil {
		var zero Model
		a.inner.StoreLoaded(key, zero, inserted)
	} else {
		a.inner.StoreLoaded(key, model.(Model), inserted)
	}
}
func (a *loaderAdapter[Input, Model]) StoreInput(key string, input any) {
	if input == nil {
		var zero Input
		a.inner.StoreInput(key, zero)
	} else {
		a.inner.StoreInput(key, input.(Input))
	}
}
func (a *loaderAdapter[Input, Model]) ModelPK(model any) map[string]any {
	return a.inner.ModelPK(model.(Model))
}
func (a *loaderAdapter[Input, Model]) Delete(ctx context.Context, key string) error {
	return a.inner.Delete(ctx, key)
}
func (a *loaderAdapter[Input, Model]) Select(ctx context.Context, key string) (any, error) {
	return a.inner.Select(ctx, key)
}

// Unwrap returns the concrete Loader held by this adapter.
func (a *loaderAdapter[Input, Model]) Unwrap() any {
	return a.inner
}

// LoadedStore provides type-erased access to loaded models.
// Concrete loaders implement this with a GetAny method (suffixed to avoid
// colliding with their typed Get methods).
type LoadedStore interface {
	GetAny(key string) (any, bool)
}

func (a *loaderAdapter[Input, Model]) Get(key string) (any, bool) {
	if store, ok := any(a.inner).(LoadedStore); ok {
		return store.GetAny(key)
	}
	return nil, false
}

// ---------------------------------------------------------------------------
// Get / MustGet — generic accessors via Fixture
// ---------------------------------------------------------------------------

// Get retrieves a loaded model by its type and fixture key.
func Get[T any](f *Fixture, key string) (T, bool) {
	loaderName, ok := loaderNameForType(f, reflect.TypeFor[T]())
	if !ok {
		var zero T
		return zero, false
	}
	loader, ok := f.loaders[loaderName]
	if !ok {
		var zero T
		return zero, false
	}
	v, ok := loader.Get(key)
	if !ok {
		var zero T
		return zero, false
	}
	return v.(T), true
}

// MustGet retrieves a loaded model or panics if not found. Intended for tests.
func MustGet[T any](f *Fixture, key string) T {
	v, ok := Get[T](f, key)
	if !ok {
		typeName := reflect.TypeFor[T]().String()
		panic(fmt.Sprintf("dbfixture: %s/%s not found", typeName, key))
	}
	return v
}

// ---------------------------------------------------------------------------
// LoaderParams — shared dependency bag for all model loaders
// ---------------------------------------------------------------------------

// LoaderParams holds all dependencies that model loaders may need.
type LoaderParams struct {
	PG *bun.DB
	bootstrap.ClickhouseParams
}
