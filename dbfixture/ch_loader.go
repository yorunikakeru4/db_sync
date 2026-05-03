package dbfixture

import (
	"context"
	"fmt"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/attrval"
	"github.com/uptrace/uptrace/pkg/hashutil"
	"github.com/uptrace/uptrace/pkg/idgen"
	"github.com/uptrace/uptrace/tracing"
)

// chSpanLoaderBase provides shared loaded storage for Span/Log/Event loaders.
type chSpanLoaderBase struct {
	params  LoaderParams
	fixture *Fixture
	name    string
	loaded  map[idgen.SpanID]*models.Span

	// idTables lists the ClickHouse index and data tables for this span type.
	// Used by Delete to remove individual records.
	idTables []string

	// assignGroupID controls whether group hashes are computed.
	assignGroupID bool
	// hasher is reused across span init calls to avoid repeated allocation.
	hasher *hashutil.Hasher
}

// ApplyOpt implements OptApplier.
func (b *chSpanLoaderBase) ApplyOpt(opt any) {
	switch o := opt.(type) {
	case assignGroupIDOpt:
		b.assignGroupID = bool(o)
		if b.assignGroupID {
			b.hasher = hashutil.NewHasher()
		}
	}
}

func (b *chSpanLoaderBase) setLoaded(id idgen.SpanID, span *models.Span) {
	if b.loaded == nil {
		b.loaded = make(map[idgen.SpanID]*models.Span)
	}
	b.loaded[id] = span
}

func (b *chSpanLoaderBase) Get(id idgen.SpanID) *models.Span { return b.loaded[id] }

func (b *chSpanLoaderBase) All() map[idgen.SpanID]*models.Span { return b.loaded }

func (b *chSpanLoaderBase) GetAny(key string) (any, bool) {
	id, err := idgen.ParseSpanID(key)
	if err != nil {
		return nil, false
	}
	span, ok := b.loaded[id]
	return span, ok
}

// Loaded returns a snapshot map of all loaded spans keyed by SpanID. The
// returned map is a fresh copy so callers may mutate it freely.
func (b *chSpanLoaderBase) Loaded() map[idgen.SpanID]*models.Span {
	result := make(map[idgen.SpanID]*models.Span, len(b.loaded))
	for id, span := range b.loaded {
		result[id] = span
	}
	return result
}

// Resolve validates the input, normalizes array attrs, and resolves project references.
// Trace IDs are expected to be pre-resolved via InitSpanTraceIDs.
func (b *chSpanLoaderBase) Resolve(ctx context.Context, input *seedinput.Span) error {
	if input.ID == 0 {
		return fmt.Errorf("span.id is required")
	}

	// Normalize array attrs.
	for key, value := range input.Attrs {
		if arr, ok := value.([]any); ok {
			input.Attrs[key] = attrval.Array(arr)
		}
	}

	project, err := resolveProject(b.fixture, input.ProjectKey)
	if err != nil {
		return err
	}
	input.ProjectID = project.ID

	if input.TraceID.IsZero() {
		input.TraceID = input.NewTraceID()
	}

	return nil
}

func (b *chSpanLoaderBase) Defaults(ctx context.Context, input *seedinput.Span, fake bool) error {
	return spanDefaults(b.fixture, input, b.assignGroupID, b.hasher)
}

// PopulateModel converts a seedinput.Span into a models.Span.
func (b *chSpanLoaderBase) PopulateModel(ctx context.Context, model *models.Span, input *seedinput.Span) ([]string, error) {
	*model = *spanFromInput(input, b.fixture)
	return nil, nil
}

func (b *chSpanLoaderBase) Validate(ctx context.Context, model *models.Span) error {
	return nil
}

func (b *chSpanLoaderBase) Insert(ctx context.Context, model *models.Span) (bool, error) {
	return false, fmt.Errorf("chSpanLoaderBase.Insert must be overridden")
}

func (b *chSpanLoaderBase) Update(ctx context.Context, model *models.Span, columns []string) error {
	return fmt.Errorf("immutable in ClickHouse")
}

// NewModel returns a zero-value span.
func (b *chSpanLoaderBase) NewModel() *models.Span { return new(models.Span) }

// StoreLoaded stores the span in the loaded map.
// If model is nil, this is a no-op.
func (b *chSpanLoaderBase) StoreLoaded(key string, model *models.Span, inserted bool) {
	if model == nil {
		return
	}
	b.setLoaded(model.ID, model)
}

// StoreInput is a no-op for ClickHouse loaders.
func (b *chSpanLoaderBase) StoreInput(key string, input *seedinput.Span) {}

// ModelPK returns the span ID as the primary key. The real ClickHouse sort
// key is (project_id, trace_id_low, trace_id_high, id), so a delete filtered
// only by id forces a full table scan — but in the fixture/test path the
// tables are tiny and this keeps both the PK map and Delete trivial. The map
// is stored in the central KeyStore so Clear can enumerate spans across runs.
func (b *chSpanLoaderBase) ModelPK(model *models.Span) map[string]any {
	return map[string]any{"id": uint64(model.ID)}
}

// Delete removes a single span record from ClickHouse by id. This scans the
// full table per delete, which is fine for fixture-sized data.
func (b *chSpanLoaderBase) Delete(ctx context.Context, key string) error {
	id, err := idgen.ParseSpanID(key)
	if err != nil {
		return fmt.Errorf("parse span id %q: %w", key, err)
	}

	db := b.params.CHForWriting()
	for _, table := range b.idTables {
		if _, err := db.NewAlterDelete().
			Table(table).
			Where("id = ?", id).
			Exec(ctx); err != nil {
			return err
		}
	}

	if err := b.fixture.keys.Delete(b.name, key); err != nil {
		return err
	}
	delete(b.loaded, id)
	return nil
}

func (b *chSpanLoaderBase) Select(ctx context.Context, key string) (*models.Span, error) {
	return nil, fmt.Errorf("built in-memory during Insert")
}

// insertSpanRecord is the shared insert logic for Span/Log/Event loaders.
// The caller passes a pre-built model (already populated by PopulateModel),
// a pre-built index, and a zero-value data record.
func (b *chSpanLoaderBase) insertSpanRecord(
	ctx context.Context,
	model *models.Span,
	index tracing.IndexRecord,
	data tracing.DataRecord,
	indexTable, dataTable string,
) (bool, error) {
	project, err := b.resolveProjectByID(model.ProjectID)
	if err != nil {
		return false, err
	}

	db := b.params.CHForWriting()
	consumerData := tracing.NewConsumerData()

	index.FillFrom(consumerData, project, model)
	data.FillFrom(model, index)

	if _, err := db.NewInsert().Model(index).DistModelTable(indexTable).Exec(ctx); err != nil {
		return false, err
	}
	if _, err := db.NewInsert().Model(data).DistModelTable(dataTable).Exec(ctx); err != nil {
		return false, err
	}
	if err := insertSpanLinks(b.params, ctx, project, model); err != nil {
		return false, err
	}

	return true, nil
}

// resolveProjectByID finds a project by ID from already-loaded fixtures.
func (b *chSpanLoaderBase) resolveProjectByID(projectID uint32) (*models.Project, error) {
	if p := b.fixture.overrideProject(); p != nil {
		return p, nil
	}
	for _, entry := range UnwrapLoader[*ProjectLoader](b.fixture, ModelProject).Loaded() {
		if entry.Value.ID == projectID {
			return entry.Value, nil
		}
	}
	return nil, fmt.Errorf("dbfixture: project with id=%d not found", projectID)
}

