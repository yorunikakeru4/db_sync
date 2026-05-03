package dbfixture

import (
	"context"
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/uptrace/bun"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/pkg/eventbus"
	"github.com/uptrace/uptrace/pkg/idgen"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// Seed loads seed data into the database within a single PG transaction.
// Each item is processed through the pipeline: Resolve → (Defaults if new) →
// Validate → Insert or Update.
//
// When seed.Update is true, existing keys are updated instead of inserted.
// When seed.Delete is true, orphaned records are deleted after loading.
func (f *Fixture) Seed(ctx context.Context, seed *seedinput.SeedData) error {
	// Pre-process: resolve trace IDs across all span inputs.
	allSpanInputs := collectSpanInputs(seed)
	if len(allSpanInputs) > 0 {
		if err := InitSpanTraceIDs(allSpanInputs); err != nil {
			return err
		}
	}

	if err := f.pg.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		ctx = bunTxKey.WithValue(ctx, &tx)

		// Load existing fixture keys into the central KeyStore.
		if f.persistKeys {
			if err := f.keys.LoadAll(ctx, f.fixtureKeys); err != nil {
				return err
			}
		}

		for _, name := range f.presetNames {
			if err := f.seedPreset(ctx, name, seed); err != nil {
				return err
			}
		}

		if err := f.seedData(ctx, seed); err != nil {
			return err
		}

		// Flush key mutations to the database.
		if f.persistKeys {
			if err := f.keys.SaveDirty(ctx, f.fixtureKeys); err != nil {
				return err
			}
			return f.keys.FlushDeleted(ctx, f.fixtureKeys)
		}
		return nil
	}); err != nil {
		return err
	}

	return f.publishEvents(ctx)
}

// seedPreset loads a built-in YAML preset from presetsFS and seeds it with
// fake data temporarily enabled so that uniqueness-sensitive fields left blank
// in the YAML (email, tokens) get populated per call.
func (f *Fixture) seedPreset(ctx context.Context, name string, parent *seedinput.SeedData) error {
	path := "presets/" + name + ".yml"
	data, err := presetsFS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("dbfixture: read preset %q: %w", name, err)
	}
	data, err = renderTemplate(data, nil, f.clock)
	if err != nil {
		return fmt.Errorf("dbfixture: render preset %q: %w", name, err)
	}
	seed := new(seedinput.SeedData)
	if err := yaml.UnmarshalWithOptions(data, seed, yaml.DisallowUnknownField()); err != nil {
		return fmt.Errorf("dbfixture: parse preset %q: %w", name, err)
	}

	// Temporarily flip fakeData so fake data runs on preset inputs regardless of
	// the fixture-wide toggle. Restore afterwards so the user seed is unaffected.
	prev := f.fakeData
	f.fakeData = true
	defer func() { f.fakeData = prev }()

	// Inherit update/delete flags from the parent seed.
	seed.Update = parent.Update
	seed.Delete = parent.Delete
	return f.seedData(ctx, seed)
}

// seedData seeds all items from the SeedData in dependency order.
func (f *Fixture) seedData(ctx context.Context, seed *seedinput.SeedData) error {
	// Track which keys are seeded in this call so deleteOrphans can detect
	// keys that existed before but are absent from the current seed.
	seededKeys := make(map[string]map[string]bool)

	seedOne := func(name string, input any) error {
		key := input.(Keyer).FixtureKey()
		if key != "" {
			m := seededKeys[name]
			if m == nil {
				m = make(map[string]bool)
				seededKeys[name] = m
			}
			m[key] = true
		}
		return f.seedOne(ctx, name, input, seed.Update)
	}

	// Process in dependency order matching Fixture registration.
	for _, org := range seed.Orgs {
		if err := seedOne(ModelOrg, org); err != nil {
			return err
		}
	}
	for _, user := range seed.Users {
		if err := seedOne(ModelUser, user); err != nil {
			return err
		}
		// Expand user.Orgs → OrgUser entries.
		for _, org := range user.Orgs {
			orgUserKey := fmt.Sprintf("%s-%s", org.OrgKey, user.Key)
			orgUser := &seedinput.OrgUser{
				Key:     orgUserKey,
				UserKey: user.Key,
				OrgKey:  org.OrgKey,
				Role:    org.Role,
			}
			if err := seedOne(ModelOrgUser, orgUser); err != nil {
				return err
			}
		}
	}
	for _, token := range seed.UserTokens {
		if err := seedOne(ModelUserToken, token); err != nil {
			return err
		}
	}
	for _, notification := range seed.UserNotifications {
		if err := seedOne(ModelUserNotification, notification); err != nil {
			return err
		}
	}
	for _, orgUser := range seed.OrgUsers {
		if err := seedOne(ModelOrgUser, orgUser); err != nil {
			return err
		}
	}
	for _, project := range seed.Projects {
		if err := seedOne(ModelProject, project); err != nil {
			return err
		}
	}
	for _, token := range seed.ProjectTokens {
		if err := seedOne(ModelProjectToken, token); err != nil {
			return err
		}
	}
	for _, user := range seed.ProjectUsers {
		if err := seedOne(ModelProjectUser, user); err != nil {
			return err
		}
	}
	// Expand user.Projects → ProjectUser entries.
	for _, user := range seed.Users {
		if len(user.Projects) > 0 && len(user.Orgs) == 0 {
			return fmt.Errorf("dbfixture: user %q has projects but no orgs", user.Key)
		}
		for _, proj := range user.Projects {
			projectUserKey := fmt.Sprintf("%s-%s", proj.ProjectKey, user.Key)
			orgUserKey := user.Orgs[0].OrgKey + "-" + user.Key
			projectUser := &seedinput.ProjectUser{
				Key:        projectUserKey,
				ProjectKey: proj.ProjectKey,
				OrgUserKey: orgUserKey,
				PermLevel:  proj.PermLevel,
			}
			if err := seedOne(ModelProjectUser, projectUser); err != nil {
				return err
			}
		}
	}
	for _, team := range seed.Teams {
		if err := seedOne(ModelTeam, team); err != nil {
			return err
		}
	}
	for _, user := range seed.TeamUsers {
		if err := seedOne(ModelTeamUser, user); err != nil {
			return err
		}
	}
	for _, project := range seed.TeamProjects {
		if err := seedOne(ModelTeamProject, project); err != nil {
			return err
		}
	}
	for _, dashboard := range seed.Dashboards {
		if err := seedOne(ModelDashboard, dashboard); err != nil {
			return err
		}
	}
	for _, tag := range seed.ProjectTags {
		if err := seedOne(ModelProjectTag, tag); err != nil {
			return err
		}
	}
	for _, taggedDash := range seed.TaggedDashboards {
		if err := seedOne(ModelTaggedDashboard, taggedDash); err != nil {
			return err
		}
	}
	for _, monitor := range seed.MetricMonitors {
		if err := seedOne(ModelMetricMonitor, monitor); err != nil {
			return err
		}
	}
	for _, monitor := range seed.ErrorMonitors {
		if err := seedOne(ModelErrorMonitor, monitor); err != nil {
			return err
		}
	}
	for _, alert := range seed.MetricAlerts {
		if err := seedOne(ModelMetricAlert, alert); err != nil {
			return err
		}
	}
	for _, alert := range seed.ErrorAlerts {
		if err := seedOne(ModelErrorAlert, alert); err != nil {
			return err
		}
	}
	for _, event := range seed.MetricAlertEvents {
		if err := seedOne(ModelMetricAlertEvent, event); err != nil {
			return err
		}
	}
	for _, event := range seed.ErrorAlertEvents {
		if err := seedOne(ModelErrorAlertEvent, event); err != nil {
			return err
		}
	}
	for _, note := range seed.AlertNotes {
		if err := seedOne(ModelAlertNote, note); err != nil {
			return err
		}
	}
	for _, assignment := range seed.AlertAssignments {
		if err := seedOne(ModelAlertAssignment, assignment); err != nil {
			return err
		}
	}

	if err := seedNotifChannels(seed, seedOne); err != nil {
		return err
	}

	for _, span := range seed.Spans {
		if err := seedOne(ModelSpan, span); err != nil {
			return err
		}
	}
	for _, log := range seed.Logs {
		if err := seedOne(ModelLog, log); err != nil {
			return err
		}
	}
	for _, event := range seed.Events {
		if err := seedOne(ModelEvent, event); err != nil {
			return err
		}
	}
	for _, dp := range seed.Datapoints {
		if err := seedOne(ModelDatapoint, dp); err != nil {
			return err
		}
	}
	for _, override := range seed.TraceOverrides {
		if err := seedOne(ModelTraceOverride, override); err != nil {
			return err
		}
	}

	// Delete orphans in reverse dependency order if enabled.
	if seed.Delete {
		if err := f.deleteOrphans(ctx, seededKeys); err != nil {
			return err
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// SeedOne
// ---------------------------------------------------------------------------

// SeedOne runs the full pipeline for a single input of the named loader.
//
// The pipeline is: Resolve → (Defaults if new) → Select-by-key →
// PopulateModel → Validate → Insert or Update → StoreLoaded/StoreInput.
func (f *Fixture) SeedOne(ctx context.Context, name string, input any) error {
	return f.seedOne(ctx, name, input, true)
}

// seedOne is the internal implementation that accepts an updates flag.
func (f *Fixture) seedOne(ctx context.Context, name string, input any, updates bool) error {
	loader, ok := f.Loader(name)
	if !ok {
		return fmt.Errorf("dbfixture: loader %q not registered", name)
	}

	key := input.(Keyer).FixtureKey()
	// pfx chains model name and fixture key onto the error so that
	// failures anywhere in the pipeline carry full context (e.g.
	// "users: alice: validate: ...") regardless of whether seedOne
	// was reached via Seed or the public SeedOne entrypoint.
	pfx := func(err error) error {
		if key != "" {
			err = validerr.Prefix(key, err)
		}
		return validerr.Prefix(name, err)
	}

	if err := loader.Resolve(ctx, input); err != nil {
		return pfx(err)
	}

	// Decide insert vs update based on whether the key is already known
	// from the central KeyStore (persisted or in-memory) or from the
	// loader's loaded map.
	existing := f.keys.Has(name, key)
	if !existing && updates && key != "" {
		_, existing = loader.Get(key)
	}

	var model any
	if existing {
		var err error
		model, err = loader.Select(ctx, key)
		if err != nil {
			// Loader doesn't support Select (e.g. append-only ClickHouse loaders).
			// Fall back to the insert path.
			existing = false
		}
	}

	// Only apply defaults for new records — updates should only touch
	// the fields explicitly provided in the input.
	if !existing {
		if err := loader.Defaults(ctx, input, f.fakeData); err != nil {
			return pfx(err)
		}
		model = loader.NewModel()
	}

	columns, err := loader.PopulateModel(ctx, model, input)
	if err != nil {
		return pfx(err)
	}

	if f.validation {
		if err := loader.Validate(ctx, model); err != nil {
			return pfx(err)
		}
	}

	if existing {
		if len(columns) > 0 {
			if err := loader.Update(ctx, model, columns); err != nil {
				return pfx(err)
			}
		}
	} else {
		if _, err := loader.Insert(ctx, model); err != nil {
			return pfx(err)
		}

		// Store key→PK mapping in the central KeyStore.
		if key != "" {
			if pk := loader.ModelPK(model); pk != nil {
				if err := f.StoreKey(name, key, pk); err != nil {
					return pfx(err)
				}
			}
		}
	}

	loader.StoreInput(key, input)
	loader.StoreLoaded(key, model, !existing)

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// deleteOrphans detects keys that existed before this seed but are absent from
// seededKeys, and calls loader.Delete for each orphan in reverse dependency order.
//
// Known keys come from the central KeyStore which accumulates both persisted
// keys (from fixture_keys) and in-memory keys (from previous seeds).
func (f *Fixture) deleteOrphans(ctx context.Context, seededKeys map[string]map[string]bool) error {
	for i := len(f.order) - 1; i >= 0; i-- {
		name := f.order[i]
		loader, ok := f.Loader(name)
		if !ok {
			return fmt.Errorf("dbfixture: loader %q not registered", name)
		}
		seeded := seededKeys[name]

		for key := range f.keys.Keys(name) {
			if !seeded[key] {
				if err := loader.Delete(ctx, key); err != nil {
					return validerr.Prefix(name+"/"+key, err)
				}
			}
		}
	}
	return nil
}

// collectSpanInputs collects all span inputs from seed data into a single map
// for trace ID pre-resolution.
func collectSpanInputs(seed *seedinput.SeedData) map[idgen.SpanID]*seedinput.Span {
	total := len(seed.Spans) + len(seed.Logs) + len(seed.Events)
	if total == 0 {
		return nil
	}
	m := make(map[idgen.SpanID]*seedinput.Span, total)
	for _, s := range seed.Spans {
		m[s.ID] = s
	}
	for _, s := range seed.Logs {
		m[s.ID] = s
	}
	for _, s := range seed.Events {
		m[s.ID] = s
	}
	return m
}

func (f *Fixture) publishEvents(ctx context.Context) error {
	if err := publishInserted(ctx, UnwrapLoader[*UserLoader](f, ModelUser), f.events.UserCreated); err != nil {
		return err
	}
	if err := publishInserted(ctx, UnwrapLoader[*OrgLoader](f, ModelOrg), f.events.OrgCreated); err != nil {
		return err
	}
	return publishInserted(ctx, UnwrapLoader[*ProjectLoader](f, ModelProject), f.events.ProjectCreated)
}

// loadedIter is implemented by loaders that provide a typed snapshot of all loaded entries.
type loadedIter[T any] interface {
	Loaded() map[string]*Entry[T]
}

// publishInserted publishes an event for each newly inserted model.
func publishInserted[T any](ctx context.Context, loader loadedIter[T], event *eventbus.Event[T]) error {
	for _, entry := range loader.Loaded() {
		if !entry.Inserted {
			continue
		}
		if err := event.Publish(ctx, entry.Value); err != nil {
			return err
		}
	}
	return nil
}
