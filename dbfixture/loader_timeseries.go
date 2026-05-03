package dbfixture

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// TimeseriesLoader handles datapoints keyed by fingerprint (one-to-many), stored in ClickHouse.
type TimeseriesLoader struct {
	params  LoaderParams
	fixture *Fixture
	loaded  map[uint64][]*models.Datapoint
}

func NewTimeseriesLoader(f *Fixture, params LoaderParams) *TimeseriesLoader {
	return &TimeseriesLoader{params: params, fixture: f}
}

func (l *TimeseriesLoader) NewInput() *seedinput.Datapoint { return new(seedinput.Datapoint) }
func (l *TimeseriesLoader) NewModel() *models.Datapoint    { return new(models.Datapoint) }

func (l *TimeseriesLoader) GetByFingerprint(fp uint64) []*models.Datapoint   { return l.loaded[fp] }
func (l *TimeseriesLoader) AllByFingerprint() map[uint64][]*models.Datapoint { return l.loaded }

func (l *TimeseriesLoader) GetAny(key string) (any, bool) {
	fp, err := strconv.ParseUint(key, 10, 64)
	if err != nil {
		return nil, false
	}
	dps, ok := l.loaded[fp]
	return dps, ok
}

func (l *TimeseriesLoader) Resolve(ctx context.Context, input *seedinput.Datapoint) error {
	return nil
}

// Defaults sets default values for time and count fields.
func (l *TimeseriesLoader) Defaults(ctx context.Context, input *seedinput.Datapoint, fake bool) error {
	if input.Time.IsZero() {
		input.Time = l.fixture.clock.Now()
	}
	if input.Count == nil {
		count := uint64(1)
		input.Count = &count
	}
	return nil
}

func (l *TimeseriesLoader) PopulateModel(ctx context.Context, model *models.Datapoint, input *seedinput.Datapoint) ([]string, error) {
	f := l.fixture
	model.Fingerprint = input.Fingerprint
	model.Metric = input.Metric
	model.Instrument = input.Instrument
	model.Unit = input.Unit
	model.LibraryName = input.LibraryName
	model.Min = input.Min
	model.Max = input.Max
	model.Sum = input.Sum
	if input.Count != nil {
		model.Count = *input.Count
	}
	model.Gauge = input.Gauge
	model.Histogram = input.Histogram
	model.AttrKeys = input.AttrKeys
	model.AttrValues = input.AttrValues
	model.Time = input.Time
	model.CumStartTime = input.Time
	if p := f.overrideProject(); p != nil {
		model.ProjectID = p.ID
	} else if input.ProjectKey != "" {
		if project, ok := Get[*models.Project](f, input.ProjectKey); ok {
			model.ProjectID = project.ID
		}
	}
	if model.CumStartTime.IsZero() {
		model.CumStartTime = model.Time
	}
	return nil, nil
}

func (l *TimeseriesLoader) Validate(ctx context.Context, model *models.Datapoint) error {
	return nil
}

// Insert inserts the datapoint and timeseries into ClickHouse.
func (l *TimeseriesLoader) Insert(ctx context.Context, model *models.Datapoint) (bool, error) {
	db := l.params.CHForWriting()

	if _, err := db.NewInsert().Model(model).DistModelTable(models.TableDatapoints.Name).Exec(ctx); err != nil {
		return false, err
	}
	ts := new(models.Timeseries)
	ts.InitFrom(model)
	if _, err := db.NewInsert().Model(ts).DistModelTable(models.TableTimeseries.Name).Exec(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (l *TimeseriesLoader) Update(ctx context.Context, model *models.Datapoint, columns []string) error {
	return fmt.Errorf("datapoints are immutable in ClickHouse")
}

func (l *TimeseriesLoader) StoreLoaded(key string, model *models.Datapoint, inserted bool) {
	if model == nil {
		return
	}
	if l.loaded == nil {
		l.loaded = make(map[uint64][]*models.Datapoint)
	}
	l.loaded[model.Fingerprint] = append(l.loaded[model.Fingerprint], model)
}

func (l *TimeseriesLoader) StoreInput(key string, input *seedinput.Datapoint) {}

// ModelPK returns the datapoint fingerprint as the primary key. The map is
// stored in the central KeyStore so Clear can enumerate timeseries to delete.
// Multiple datapoints share a fingerprint and therefore the same fixture key.
func (l *TimeseriesLoader) ModelPK(model *models.Datapoint) map[string]any {
	return map[string]any{"fingerprint": model.Fingerprint}
}

// Delete removes all datapoints for a fingerprint extracted from the composite key.
func (l *TimeseriesLoader) Delete(ctx context.Context, key string) error {
	// Parse fingerprint from "fingerprint:time" composite key produced by AllLoaded().
	fpStr := key
	if sep := strings.IndexByte(key, ':'); sep >= 0 {
		fpStr = key[:sep]
	}

	fp, err := strconv.ParseUint(fpStr, 10, 64)
	if err != nil {
		return fmt.Errorf("parse fingerprint from %q: %w", key, err)
	}

	// Skip if already deleted by a prior key with the same fingerprint.
	if _, ok := l.loaded[fp]; !ok {
		return nil
	}

	db := l.params.CHForWriting()
	for _, table := range []string{models.TableTimeseries.Name, models.TableDatapoints.Name} {
		if _, err := db.NewAlterDelete().
			Table(table).
			Where("fingerprint = ?", fp).
			Exec(ctx); err != nil {
			return err
		}
	}

	delete(l.loaded, fp)
	return nil
}

func (l *TimeseriesLoader) Select(ctx context.Context, key string) (*models.Datapoint, error) {
	return nil, fmt.Errorf("datapoints are built in-memory during Insert")
}
