package dbfixture

import (
	"fmt"
	"iter"
	"reflect"
	"strconv"

	"github.com/uptrace/bun"
	"go.uber.org/fx"

	"github.com/uptrace/uptrace/bootstrap"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/idgen"
	"github.com/uptrace/uptrace/pkg/json"
	"github.com/uptrace/uptrace/pkg/unixtime"
)

// FixtureParams holds the dependencies required for creating fixtures.
type FixtureParams struct {
	fx.In

	PG *bun.DB
	bootstrap.ClickhouseParams

	Events *models.Events
}

// Fixture defines test fixture data to be loaded into the database.
// It holds a registry of model loaders (one per model type), the options
// that control the loading pipeline, and the database connections needed
// for seeding.
type Fixture struct {
	// Database connections and events.
	pg                  *bun.DB
	chParams            bootstrap.ClickhouseParams
	events *models.Events

	// loaders maps loader names to their AnyLoader implementations.
	loaders map[string]AnyLoader
	// order preserves registration order, which defines dependency/load order.
	order []string
	// loaderNamesByType maps model types to their unique loader names.
	loaderNamesByType map[reflect.Type]string
	// ambiguousTypes tracks model types served by multiple loaders.
	ambiguousTypes map[reflect.Type]struct{}

	// clock provides time values for created/updated timestamps.
	clock unixtime.Clock
	// fakeClock is set when WithFakeClock is used. Exposed via Clock() so
	// test commands can read the epoch and advance the clock per directive.
	fakeClock *unixtime.FakeClock

	// Pipeline toggles — read by Fixture.SeedOne.
	fakeData   bool // pass fake=true to Defaults
	validation bool // apply Validate before Insert

	// persistKeys enables loading/saving keys from the fixture_keys table.
	persistKeys bool

	// presetNames lists built-in presets (from presetsFS) to seed before the
	// user-provided seed data. Presets are applied in argument order.
	presetNames []string

	// overrideProjectKey forces all project-scoped objects to use the project
	// registered in the KeyStore under this key. Resolved lazily — the key must
	// be loaded before any project-scoped operation runs.
	overrideProjectKey string

	// fixtureKeys provides CRUD operations on the fixture_keys table.
	fixtureKeys *FixtureKeyRepo

	// keys is the central registry of fixture key → PK JSON mappings.
	// Replaces per-loader pkByKey, knownKeys, and pendingKeys.
	keys KeyStore

	// loaderOpts stores loader-specific options.
	// Every loader receives all options during Register and ignores unsupported ones.
	loaderOpts []any
}

// NewFixture creates a new Fixture with all model loaders registered in dependency order.
func NewFixture(params FixtureParams, opts ...Option) *Fixture {
	f := &Fixture{
		pg:                  params.PG,
		chParams:            params.ClickhouseParams,
		events: params.Events,

		loaders:           make(map[string]AnyLoader),
		loaderNamesByType: make(map[reflect.Type]string),
		ambiguousTypes:    make(map[reflect.Type]struct{}),
		fixtureKeys:       NewFixtureKeyRepo(params.PG),
		clock:             unixtime.NewRealClock(),
		validation:        true,
	}

	for _, opt := range opts {
		opt(f)
	}

	p := LoaderParams{
		PG:               params.PG,
		ClickhouseParams: params.ClickhouseParams,
	}

	// Registration order = dependency order = load order.
	f.Register(ModelOrg, Wrap(NewOrgLoader(f, p)))
	f.Register(ModelUser, Wrap(NewUserLoader(f, p)))
	f.Register(ModelUserToken, Wrap(NewUserTokenLoader(f, p)))
	f.Register(ModelUserNotification, Wrap(NewUserNotificationLoader(f, p)))
	f.Register(ModelOrgUser, Wrap(NewOrgUserLoader(f, p)))
	f.Register(ModelProject, Wrap(NewProjectLoader(f, p)))
	f.Register(ModelProjectToken, Wrap(NewProjectTokenLoader(f, p)))
	f.Register(ModelProjectUser, Wrap(NewProjectUserLoader(f, p)))
	f.Register(ModelTeam, Wrap(NewTeamLoader(f, p)))
	f.Register(ModelTeamUser, Wrap(NewTeamUserLoader(f, p)))
	f.Register(ModelTeamProject, Wrap(NewTeamProjectLoader(f, p)))
	f.Register(ModelDashboard, Wrap(NewDashboardLoader(f, p)))
	f.Register(ModelProjectTag, Wrap(NewProjectTagLoader(f, p)))
	f.Register(ModelTaggedDashboard, Wrap(NewTaggedDashboardLoader(f, p)))
	f.Register(ModelMetricMonitor, Wrap(NewMetricMonitorLoader(f, p)))
	f.Register(ModelErrorMonitor, Wrap(NewErrorMonitorLoader(f, p)))
	f.Register(ModelMetricAlert, Wrap(NewMetricAlertLoader(f, p)))
	f.Register(ModelErrorAlert, Wrap(NewErrorAlertLoader(f, p)))
	f.Register(ModelMetricAlertEvent, Wrap(NewMetricAlertEventLoader(f, p)))
	f.Register(ModelErrorAlertEvent, Wrap(NewErrorAlertEventLoader(f, p)))
	f.Register(ModelAlertNote, Wrap(NewAlertNoteLoader(f, p)))
	f.Register(ModelAlertAssignment, Wrap(NewAlertAssignmentLoader(f, p)))
	registerNotifChannelLoaders(f, p)
	f.Register(ModelLogGroupingRule, Wrap(NewLogGroupingRuleLoader(f, p)))
	f.Register(ModelSpan, Wrap(NewSpanLoader(f, p)))
	f.Register(ModelLog, Wrap(NewLogLoader(f, p)))
	f.Register(ModelEvent, Wrap(NewEventLoader(f, p)))
	f.Register(ModelTraceOverride, Wrap(NewTraceOverrideLoader(f, p)))
	f.Register(ModelDatapoint, Wrap(NewTimeseriesLoader(f, p)))

	return f
}

// Option configures a Fixture.
type Option func(*Fixture)

// WithClock sets the clock used for timestamps.
func WithClock(clock unixtime.Clock) Option {
	return func(f *Fixture) {
		f.clock = clock
	}
}

// WithFakeClock installs a test fake clock. It sets both the general
// timestamp clock (same as WithClock) and exposes the fake via Fixture.Clock
// so test commands can read the epoch and advance the clock per directive.
func WithFakeClock(fc *unixtime.FakeClock) Option {
	return func(f *Fixture) {
		f.clock = fc
		f.fakeClock = fc
	}
}

// Clock returns the fake clock installed via WithFakeClock, or nil if the
// fixture was built without one. Test commands use this to read f.Clock().
// Epoch() for query time bounds and f.Clock().Set/Adjust to move the clock.
func (f *Fixture) Clock() *unixtime.FakeClock {
	return f.fakeClock
}

// WithPresets registers built-in YAML presets (from dbfixture/presets) to seed
// before the user-provided seed data. Presets are applied in argument order
// within the same transaction. Presets run with FakeData enabled so that
// uniqueness-sensitive fields left blank in the YAML (user email, user/project
// tokens) get populated per seed call — preventing UNIQUE collisions across
// reseeds while preserving stable identity via fixture keys.
func WithPresets(names ...string) Option {
	return func(f *Fixture) {
		f.presetNames = append(f.presetNames, names...)
	}
}

// WithFakeData enables random fake data population in Defaults.
func WithFakeData(on bool) Option {
	return func(f *Fixture) {
		f.fakeData = on
	}
}

// WithValidation controls whether Validate is called before Insert.
func WithValidation(on bool) Option {
	return func(f *Fixture) {
		f.validation = on
	}
}

// WithPersistKeys controls whether fixture keys are loaded from and saved to
// the fixture_keys table. When disabled, Seed always uses the insert path.
func WithPersistKeys(on bool) Option {
	return func(f *Fixture) {
		f.persistKeys = on
	}
}

// confirmedEmailsOpt is a loader option for the user loader.
type confirmedEmailsOpt bool

// AllowConfirmedEmails allows the email_confirmed field on users.
func AllowConfirmedEmails(on bool) Option {
	return func(f *Fixture) {
		f.addLoaderOpt(confirmedEmailsOpt(on))
	}
}

// WithProjectOverride forces all project-scoped objects to use the project
// registered in the KeyStore under projectKey. The key is resolved lazily, so
// the project must be loaded into the fixture before any project-scoped
// command executes.
func WithProjectOverride(projectKey string) Option {
	return func(f *Fixture) {
		f.overrideProjectKey = projectKey
	}
}

// overrideProject returns the override project resolved from the KeyStore, or
// nil when no override is configured or the key is not yet registered.
// The nil-on-miss behavior is required to avoid a chicken-and-egg during the
// seed that creates the override target itself: loaders call overrideProject
// while seeding the very project the override key points at.
func (f *Fixture) overrideProject() *models.Project {
	if f.overrideProjectKey == "" {
		return nil
	}
	project, _ := Get[*models.Project](f, f.overrideProjectKey)
	return project
}

// assignGroupIDOpt is a loader option for span/log/event loaders.
type assignGroupIDOpt bool

// WithAssignGroupID enables automatic group hash computation for spans.
func WithAssignGroupID(on bool) Option {
	return func(f *Fixture) {
		f.addLoaderOpt(assignGroupIDOpt(on))
	}
}

// ---------------------------------------------------------------------------
// Loader options
// ---------------------------------------------------------------------------

// addLoaderOpt appends a loader-specific option.
// All options are passed to every loader during Register;
// loaders ignore options they don't understand.
func (f *Fixture) addLoaderOpt(opt any) {
	f.loaderOpts = append(f.loaderOpts, opt)
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

// Register adds a model loader under a name. Registration order defines load order
// (dependency order). Each loader may only reference models from loaders
// registered before it.
//
// All loader options are passed to every loader that implements OptApplier;
// loaders ignore options they don't understand.
func (f *Fixture) Register(name string, loader AnyLoader) {
	if _, exists := f.loaders[name]; exists {
		panic(fmt.Sprintf("dbfixture: loader %q already registered", name))
	}
	f.loaders[name] = loader
	f.order = append(f.order, name)
	f.registerLoaderType(name, loader)

	if len(f.loaderOpts) == 0 {
		return
	}

	var applier OptApplier
	if u, ok := loader.(interface{ Unwrap() any }); ok {
		applier, _ = u.Unwrap().(OptApplier)
	}
	if applier == nil {
		return
	}
	for _, opt := range f.loaderOpts {
		applier.ApplyOpt(opt)
	}
}

func (f *Fixture) registerLoaderType(name string, loader AnyLoader) {
	modelTyper, ok := loader.(modelTyper)
	if !ok {
		return
	}

	modelType := modelTyper.ModelType()
	if _, ok := f.ambiguousTypes[modelType]; ok {
		return
	}
	if _, ok := f.loaderNamesByType[modelType]; ok {
		delete(f.loaderNamesByType, modelType)
		f.ambiguousTypes[modelType] = struct{}{}
		return
	}
	f.loaderNamesByType[modelType] = name
}

// LoaderNames returns registered loader names in dependency order.
func (f *Fixture) LoaderNames() []string {
	return f.order
}

// Loader returns a loader by name.
func (f *Fixture) Loader(name string) (AnyLoader, bool) {
	l, ok := f.loaders[name]
	return l, ok
}

// UnwrapLoader returns the concrete typed loader stored under the given name.
// Panics if the name is not registered or the type assertion fails.
func UnwrapLoader[T any](f *Fixture, name string) T {
	loader := f.loaders[name]
	return loader.(interface{ Unwrap() any }).Unwrap().(T)
}

// AllSpans returns an iterator over every loaded span across the Span, Log,
// and Event loaders.
func AllSpans(f *Fixture) iter.Seq[*models.Span] {
	return func(yield func(*models.Span) bool) {
		for _, span := range UnwrapLoader[*SpanLoader](f, ModelSpan).Loaded() {
			if !yield(span) {
				return
			}
		}
		for _, span := range UnwrapLoader[*LogLoader](f, ModelLog).Loaded() {
			if !yield(span) {
				return
			}
		}
		for _, span := range UnwrapLoader[*EventLoader](f, ModelEvent).Loaded() {
			if !yield(span) {
				return
			}
		}
	}
}

// StoreKey stores a PK mapping in the central KeyStore.
// Called after Insert to record the newly assigned PK so that subsequent
// Select calls can locate the row by its fixture key.
func (f *Fixture) StoreKey(model, key string, pk map[string]any) error {
	pkJSON, err := json.Marshal(pk)
	if err != nil {
		return fmt.Errorf("dbfixture: marshal pk for %s/%s: %w", model, key, err)
	}
	f.keys.Set(model, key, pkJSON)
	return nil
}

// GetSpan returns a loaded span by SpanID.
func (f *Fixture) GetSpan(id idgen.SpanID) *models.Span {
	v, _ := f.loaders[ModelSpan].Get(SpanIDKey(id))
	if v == nil {
		return nil
	}
	return v.(*models.Span)
}

// GetLog returns a loaded log by SpanID.
func (f *Fixture) GetLog(id idgen.SpanID) *models.Span {
	v, _ := f.loaders[ModelLog].Get(SpanIDKey(id))
	if v == nil {
		return nil
	}
	return v.(*models.Span)
}

// GetEvent returns a loaded event by SpanID.
func (f *Fixture) GetEvent(id idgen.SpanID) *models.Span {
	v, _ := f.loaders[ModelEvent].Get(SpanIDKey(id))
	if v == nil {
		return nil
	}
	return v.(*models.Span)
}

// MustGetSpan returns a loaded span by SpanID or panics.
func (f *Fixture) MustGetSpan(id idgen.SpanID) *models.Span {
	span := f.GetSpan(id)
	if span == nil {
		panic(fmt.Sprintf("dbfixture: span/%s not found", SpanIDKey(id)))
	}
	return span
}

// MustGetLog returns a loaded log by SpanID or panics.
func (f *Fixture) MustGetLog(id idgen.SpanID) *models.Span {
	log := f.GetLog(id)
	if log == nil {
		panic(fmt.Sprintf("dbfixture: log/%s not found", SpanIDKey(id)))
	}
	return log
}

// MustGetEvent returns a loaded event by SpanID or panics.
func (f *Fixture) MustGetEvent(id idgen.SpanID) *models.Span {
	event := f.GetEvent(id)
	if event == nil {
		panic(fmt.Sprintf("dbfixture: event/%s not found", SpanIDKey(id)))
	}
	return event
}

// GetTimeseries returns loaded datapoints by fingerprint.
func (f *Fixture) GetTimeseries(fingerprint uint64) []*models.Datapoint {
	v, ok := f.loaders[ModelDatapoint].Get(strconv.FormatUint(fingerprint, 10))
	if !ok {
		return nil
	}
	return v.([]*models.Datapoint)
}

func loaderNameForType(f *Fixture, modelType reflect.Type) (string, bool) {
	if _, ok := f.ambiguousTypes[modelType]; ok {
		panic(fmt.Sprintf("dbfixture: type %s is served by multiple loaders; use a loader-specific accessor", modelType))
	}
	name, ok := f.loaderNamesByType[modelType]
	return name, ok
}
