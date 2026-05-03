package dbfixture

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/brianvoe/gofakeit/v5"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/madalarm"
	"github.com/uptrace/uptrace/tsmetric/metricname"
	"github.com/uptrace/uptrace/tsmetric/mql"
)

// ---------------------------------------------------------------------------
// MetricMonitorLoader
// ---------------------------------------------------------------------------

// MetricMonitorLoader loads metric monitor fixtures.
type MetricMonitorLoader struct {
	pgLoaderBase[seedinput.MetricMonitor, models.MetricMonitor]
}

// NewMetricMonitorLoader creates a Loader for metric monitors.
func NewMetricMonitorLoader(f *Fixture, params LoaderParams) *MetricMonitorLoader {
	return &MetricMonitorLoader{pgLoaderBase: pgLoaderBase[seedinput.MetricMonitor, models.MetricMonitor]{
		params: params, fixture: f, name: ModelMetricMonitor,
	}}
}

func (l *MetricMonitorLoader) Resolve(ctx context.Context, input *seedinput.MetricMonitor) error {
	return resolveProjectID(l.fixture, &input.ProjectID, input.ProjectKey)
}

func (l *MetricMonitorLoader) Defaults(ctx context.Context, input *seedinput.MetricMonitor, fake bool) error {
	if err := initBaseMonitor(l.fixture, &input.BaseMonitor, fake); err != nil {
		return err
	}
	input.Type = models.MonitorMetric

	return nil
}

func (l *MetricMonitorLoader) NewModel() *models.MetricMonitor {
	return models.NewMetricMonitor()
}

func (l *MetricMonitorLoader) PopulateModel(ctx context.Context, model *models.MetricMonitor, input *seedinput.MetricMonitor) ([]string, error) {
	cols := populateBaseMonitorModel(model.BaseMonitor, &input.BaseMonitor)
	if input.Params != nil {
		// Parse MetricExprs and QueryParts into their model representations.
		// This runs on both insert and update paths so derived fields stay in sync.
		metrics, err := mql.ParseMetrics(input.Params.MetricExprs)
		if err != nil {
			return nil, err
		}
		parsedMetrics := make([]mql.MetricAlias, len(metrics))
		for i, metric := range metrics {
			parsedMetrics[i] = mql.MetricAlias{
				Name:  metric.Name,
				Alias: metric.Alias,
			}
		}

		model.Params = models.MetricMonitorParams{
			Metrics:       parsedMetrics,
			Query:         mql.JoinQuery(input.Params.QueryParts),
			Column: models.Column{
				Name: input.Params.Column,
				Unit: input.Params.ColumnUnit,
			},
			Resolution:    input.Params.Resolution,
			NumEvalPoints: input.Params.NumEvalPoints,
			AbsentPoints:  input.Params.AbsentPoints,
			TimeOffset:    input.Params.TimeOffset,
		}

		if input.Params.Detector.Type != "" {
			// New format: build from Detector field.
			switch input.Params.Detector.Type {
			case madalarm.DetectorManual:
				model.Params.Detector.Type = madalarm.DetectorManual
				if err := model.Params.Detector.SetParams(&madalarm.ManualDetectorParams{
					MinValue: input.Params.Detector.MinValue,
					MaxValue: input.Params.Detector.MaxValue,
					Recovery: madalarm.Recovery{
						MinValue: input.Params.Detector.Recovery.MinValue,
						MaxValue: input.Params.Detector.Recovery.MaxValue,
					},
				}); err != nil {
					return nil, err
				}
			case madalarm.DetectorAuto:
				model.Params.Detector.Type = madalarm.DetectorAuto
				if err := model.Params.Detector.SetParams(&madalarm.AutoDetectorParams{
					Tolerance:      input.Params.Detector.Tolerance,
					TrainingPeriod: input.Params.Detector.TrainingPeriod,
					MinDevFraction: input.Params.Detector.MinDevFraction,
					MinDevAbsolute: input.Params.Detector.MinDevAbsolute,
				}); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unsupported detector type: %q", input.Params.Detector.Type)
			}
		} else {
			// Deprecated format: flat fields.
			switch input.Params.BoundsSource {
			case madalarm.BoundsSourceManual, "":
				model.Params.Detector.Type = madalarm.DetectorManual
				if err := model.Params.Detector.SetParams(&madalarm.ManualDetectorParams{
					MinValue: input.Params.MinAllowedValue,
					MaxValue: input.Params.MaxAllowedValue,
					Recovery: madalarm.Recovery{
						MinValue: input.Params.Flapping.MinAllowedValue,
						MaxValue: input.Params.Flapping.MaxAllowedValue,
					},
				}); err != nil {
					return nil, err
				}
			case madalarm.BoundsSourceAuto:
				model.Params.Detector.Type = madalarm.DetectorAuto
				if err := model.Params.Detector.SetParams(&madalarm.AutoDetectorParams{
					Tolerance:      input.Params.Tolerance,
					TrainingPeriod: input.Params.TrainingPeriod,
					MinDevFraction: input.Params.MinDevFraction,
					MinDevAbsolute: input.Params.MinDevAbsolute,
				}); err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("unsupported bounds source: %q", input.Params.BoundsSource)
			}
		}

		cols = append(cols, "params")
	}
	return cols, nil
}

// ---------------------------------------------------------------------------
// ErrorMonitorLoader
// ---------------------------------------------------------------------------

// ErrorMonitorLoader loads error monitor fixtures.
type ErrorMonitorLoader struct {
	pgLoaderBase[seedinput.ErrorMonitor, models.ErrorMonitor]
}

// NewErrorMonitorLoader creates a Loader for error monitors.
func NewErrorMonitorLoader(f *Fixture, params LoaderParams) *ErrorMonitorLoader {
	return &ErrorMonitorLoader{pgLoaderBase: pgLoaderBase[seedinput.ErrorMonitor, models.ErrorMonitor]{
		params: params, fixture: f, name: ModelErrorMonitor,
	}}
}

func (l *ErrorMonitorLoader) Resolve(ctx context.Context, input *seedinput.ErrorMonitor) error {
	return resolveProjectID(l.fixture, &input.ProjectID, input.ProjectKey)
}

func (l *ErrorMonitorLoader) Defaults(ctx context.Context, input *seedinput.ErrorMonitor, fake bool) error {
	if err := initBaseMonitor(l.fixture, &input.BaseMonitor, fake); err != nil {
		return err
	}
	input.Type = models.MonitorError

	return nil
}

func (l *ErrorMonitorLoader) NewModel() *models.ErrorMonitor {
	return models.NewErrorMonitor()
}

func (l *ErrorMonitorLoader) PopulateModel(ctx context.Context, model *models.ErrorMonitor, input *seedinput.ErrorMonitor) ([]string, error) {
	cols := populateBaseMonitorModel(model.BaseMonitor, &input.BaseMonitor)
	if input.Params != nil {
		model.Params = models.ErrorMonitorParams{
			Metrics: []mql.MetricAlias{
				{
					Name:  metricname.UptraceTracingLogs,
					Alias: "$logs",
				},
			},
			Query: mql.JoinQuery(input.Params.QueryParts),
		}
		cols = append(cols, "params")
	}
	return cols, nil
}

// ---------------------------------------------------------------------------
// Monitor helpers
// ---------------------------------------------------------------------------

func initBaseMonitor(f *Fixture, monitor *seedinput.BaseMonitor, fake bool) error {
	monitor.CreatedAt = f.clock.Now()
	monitor.UpdatedAt = monitor.CreatedAt

	if p := f.overrideProject(); p != nil {
		monitor.ProjectID = p.ID
	}

	if monitor.RepeatIntervalParams != nil {
		strategy, err := models.NewRepeatStrategy(*monitor.RepeatIntervalParams)
		if err != nil {
			return err
		}
		monitor.RepeatInterval = &models.RepeatInterval{
			RepeatStrategy: strategy,
		}
	}

	if fake {
		if monitor.Name == nil || *monitor.Name == "" {
			name := gofakeit.Name()
			monitor.Name = &name
		}
		if monitor.Status == nil || *monitor.Status == "" {
			statuses := []models.MonitorStatus{
				models.MonitorActive,
				models.MonitorPaused,
				models.MonitorDisabled,
			}
			status := statuses[rand.IntN(len(statuses))]
			monitor.Status = &status
		}
		if monitor.TrendAggFunc == nil || *monitor.TrendAggFunc == "" {
			aggs := []models.TrendAggFuncName{models.TrendAggSum, models.TrendAggAvg, models.TrendAggMedian, models.TrendAggLast}
			agg := aggs[rand.IntN(len(aggs))]
			monitor.TrendAggFunc = &agg
		}
		if monitor.NotifyEveryoneByEmail == nil {
			notify := gofakeit.Bool()
			monitor.NotifyEveryoneByEmail = &notify
		}
	}

	return nil
}

// populateBaseMonitorModel maps seedinput.BaseMonitor fields to models.BaseMonitor
// and returns the column names that were set.
func populateBaseMonitorModel(model *models.BaseMonitor, input *seedinput.BaseMonitor) []string {
	var cols []string
	if input.ProjectID != 0 && model.ProjectID != input.ProjectID {
		model.ProjectID = input.ProjectID
		cols = append(cols, "project_id")
	}
	if input.Key != "" && model.Key != input.Key {
		model.Key = input.Key
		cols = append(cols, "key")
	}
	if input.Name != nil && model.Name != *input.Name {
		model.Name = *input.Name
		cols = append(cols, "name")
	}
	if input.IsNameTemplated && model.IsNameTemplated != input.IsNameTemplated {
		model.IsNameTemplated = input.IsNameTemplated
		cols = append(cols, "is_name_templated")
	}
	if input.Type != "" && model.Type != input.Type {
		model.Type = input.Type
		cols = append(cols, "type")
	}
	if input.Status != nil && model.Status != *input.Status {
		model.Status = *input.Status
		cols = append(cols, "status")
	}
	if input.TrendAggFunc != nil && model.TrendAggFunc != *input.TrendAggFunc {
		model.TrendAggFunc = *input.TrendAggFunc
		cols = append(cols, "trend_agg_func")
	}
	if input.TrendSensitivity != nil && model.TrendSensitivity != *input.TrendSensitivity {
		model.TrendSensitivity = *input.TrendSensitivity
		cols = append(cols, "trend_sensitivity")
	}
	if input.RepeatInterval != nil {
		model.RepeatInterval = *input.RepeatInterval
		cols = append(cols, "repeat_interval")
	}
	if input.NotifyEveryoneByEmail != nil && model.NotifyEveryoneByEmail != *input.NotifyEveryoneByEmail {
		model.NotifyEveryoneByEmail = *input.NotifyEveryoneByEmail
		cols = append(cols, "notify_everyone_by_email")
	}
	if !input.CreatedAt.IsZero() && model.CreatedAt != input.CreatedAt {
		model.CreatedAt = input.CreatedAt
		cols = append(cols, "created_at")
	}
	if !input.UpdatedAt.IsZero() && model.UpdatedAt != input.UpdatedAt {
		model.UpdatedAt = input.UpdatedAt
		cols = append(cols, "updated_at")
	}
	return cols
}
