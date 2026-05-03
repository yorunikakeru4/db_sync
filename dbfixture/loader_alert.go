package dbfixture

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/uptrace/bun"
	"github.com/zeebo/xxh3"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/tsmetric/mql"
)

// ---------------------------------------------------------------------------
// MetricAlertLoader
// ---------------------------------------------------------------------------

// MetricAlertLoader loads metric alert fixtures.
type MetricAlertLoader struct {
	pgLoaderBase[seedinput.MetricAlert, models.MetricAlert]
}

// NewMetricAlertLoader creates a Loader for metric alerts.
func NewMetricAlertLoader(f *Fixture, params LoaderParams) *MetricAlertLoader {
	return &MetricAlertLoader{pgLoaderBase: pgLoaderBase[seedinput.MetricAlert, models.MetricAlert]{
		params: params, fixture: f, name: ModelMetricAlert,
	}}
}

func (l *MetricAlertLoader) Resolve(ctx context.Context, input *seedinput.MetricAlert) error {
	monitor, ok := Get[*models.MetricMonitor](l.fixture, input.MonitorKey)
	if !ok {
		return fmt.Errorf("metric_monitor %q not found", input.MonitorKey)
	}
	input.MonitorID = monitor.ID
	input.ProjectID = monitor.ProjectID
	return nil
}

func (l *MetricAlertLoader) Defaults(ctx context.Context, input *seedinput.MetricAlert, fake bool) error {
	monitor, ok := Get[*models.MetricMonitor](l.fixture, input.MonitorKey)
	if !ok {
		return fmt.Errorf("metric_monitor %q not found", input.MonitorKey)
	}

	input.Type = models.AlertMetric
	if err := initBaseAlert(l.fixture, monitor, &input.BaseAlert, fake); err != nil {
		return err
	}
	return nil
}

func (l *MetricAlertLoader) PopulateModel(ctx context.Context, model *models.MetricAlert, input *seedinput.MetricAlert) ([]string, error) {
	return populateBaseAlertModel(&model.BaseAlert, &input.BaseAlert), nil
}

func (l *MetricAlertLoader) Insert(ctx context.Context, model *models.MetricAlert) (bool, error) {
	db := l.DB(ctx)

	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	// Select the base alert by natural key.
	base := new(models.BaseAlert)
	if err := db.NewSelect().Model(base).
		Where("a.type = ?", model.Type).
		Where("a.project_id = ?", model.ProjectID).
		Where("a.monitor_id = ?", model.MonitorID).
		Where("a.attrs_hash = ?", model.AttrsHash).
		Scan(ctx); err != nil {
		return false, err
	}

	// Re-populate model from the selected base.
	full, err := l.selectMetricAlert(ctx, db, base.ID)
	if err != nil {
		return false, err
	}
	*model = *full
	return n != 0, nil
}

func (l *MetricAlertLoader) selectMetricAlert(
	ctx context.Context, db bun.IDB, id uint64,
) (*models.MetricAlert, error) {
	base := new(models.BaseAlert)
	if err := db.NewSelect().Model(base).
		Relation("Event").
		Where("a.id = ?", id).
		Scan(ctx); err != nil {
		return nil, err
	}
	alert := models.NewMetricAlert(base)
	if base.Event != nil {
		if err := base.Event.Params.Decode(&alert.Event.Params); err != nil {
			return nil, err
		}
	}
	return alert, nil
}

func (l *MetricAlertLoader) Select(ctx context.Context, key string) (*models.MetricAlert, error) {
	id, ok := l.pkID(key)
	if !ok {
		return nil, fmt.Errorf("dbfixture: %s/%s fixture key not found (not loaded or deleted)", l.name, key)
	}
	model, err := l.selectMetricAlert(ctx, l.DB(ctx), id)
	if err != nil {
		return nil, err
	}
	l.StoreLoaded(key, model, false)
	return model, nil
}

func (l *MetricAlertLoader) Update(ctx context.Context, model *models.MetricAlert, columns []string) error {
	return l.pgLoaderBase.Update(ctx, model, columns)
}

// ---------------------------------------------------------------------------
// ErrorAlertLoader
// ---------------------------------------------------------------------------

// ErrorAlertLoader loads error alert fixtures.
type ErrorAlertLoader struct {
	pgLoaderBase[seedinput.ErrorAlert, models.ErrorAlert]
}

// NewErrorAlertLoader creates a Loader for error alerts.
func NewErrorAlertLoader(f *Fixture, params LoaderParams) *ErrorAlertLoader {
	return &ErrorAlertLoader{pgLoaderBase: pgLoaderBase[seedinput.ErrorAlert, models.ErrorAlert]{
		params: params, fixture: f, name: ModelErrorAlert,
	}}
}

func (l *ErrorAlertLoader) Resolve(ctx context.Context, input *seedinput.ErrorAlert) error {
	monitor, ok := Get[*models.ErrorMonitor](l.fixture, input.MonitorKey)
	if !ok {
		return fmt.Errorf("error_monitor %q not found", input.MonitorKey)
	}
	input.MonitorID = monitor.ID
	input.ProjectID = monitor.ProjectID
	return nil
}

func (l *ErrorAlertLoader) Defaults(ctx context.Context, input *seedinput.ErrorAlert, fake bool) error {
	monitor, ok := Get[*models.ErrorMonitor](l.fixture, input.MonitorKey)
	if !ok {
		return fmt.Errorf("error_monitor %q not found", input.MonitorKey)
	}

	input.Type = models.AlertError
	if err := initBaseAlert(l.fixture, monitor, &input.BaseAlert, fake); err != nil {
		return err
	}
	return nil
}

func (l *ErrorAlertLoader) PopulateModel(ctx context.Context, model *models.ErrorAlert, input *seedinput.ErrorAlert) ([]string, error) {
	return populateBaseAlertModel(&model.BaseAlert, &input.BaseAlert), nil
}

func (l *ErrorAlertLoader) Insert(ctx context.Context, model *models.ErrorAlert) (bool, error) {
	db := l.DB(ctx)

	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}

	base := new(models.BaseAlert)
	if err := db.NewSelect().Model(base).
		Where("a.type = ?", model.Type).
		Where("a.project_id = ?", model.ProjectID).
		Where("a.monitor_id = ?", model.MonitorID).
		Where("a.attrs_hash = ?", model.AttrsHash).
		Scan(ctx); err != nil {
		return false, err
	}

	full, err := l.selectErrorAlert(ctx, db, base.ID)
	if err != nil {
		return false, err
	}
	*model = *full
	return n != 0, nil
}

func (l *ErrorAlertLoader) selectErrorAlert(
	ctx context.Context, db bun.IDB, id uint64,
) (*models.ErrorAlert, error) {
	base := new(models.BaseAlert)
	if err := db.NewSelect().Model(base).
		Relation("Event").
		Where("a.id = ?", id).
		Scan(ctx); err != nil {
		return nil, err
	}
	alert := models.NewErrorAlert(base)
	if base.Event != nil {
		if err := base.Event.Params.Decode(&alert.Event.Params); err != nil {
			return nil, err
		}
	}
	return alert, nil
}

func (l *ErrorAlertLoader) Select(ctx context.Context, key string) (*models.ErrorAlert, error) {
	id, ok := l.pkID(key)
	if !ok {
		return nil, fmt.Errorf("dbfixture: %s/%s fixture key not found (not loaded or deleted)", l.name, key)
	}
	model, err := l.selectErrorAlert(ctx, l.DB(ctx), id)
	if err != nil {
		return nil, err
	}
	l.StoreLoaded(key, model, false)
	return model, nil
}

func (l *ErrorAlertLoader) Update(ctx context.Context, model *models.ErrorAlert, columns []string) error {
	return l.pgLoaderBase.Update(ctx, model, columns)
}

// ---------------------------------------------------------------------------
// MetricAlertEventLoader
// ---------------------------------------------------------------------------

// MetricAlertEventLoader loads metric alert event fixtures.
type MetricAlertEventLoader struct {
	pgLoaderBase[seedinput.MetricAlertEvent, models.MetricAlertEvent]
}

// NewMetricAlertEventLoader creates a Loader for metric alert events.
func NewMetricAlertEventLoader(f *Fixture, params LoaderParams) *MetricAlertEventLoader {
	return &MetricAlertEventLoader{pgLoaderBase: pgLoaderBase[seedinput.MetricAlertEvent, models.MetricAlertEvent]{
		params: params, fixture: f, name: ModelMetricAlertEvent,
	}}
}

func (l *MetricAlertEventLoader) Resolve(ctx context.Context, input *seedinput.MetricAlertEvent) error {
	alert, ok := Get[*models.MetricAlert](l.fixture, input.AlertKey)
	if !ok {
		return fmt.Errorf("metric_alert %q not found", input.AlertKey)
	}
	input.AlertID = alert.ID
	input.ProjectID = alert.ProjectID
	return nil
}

func (l *MetricAlertEventLoader) Defaults(ctx context.Context, input *seedinput.MetricAlertEvent, fake bool) error {
	alert, ok := Get[*models.MetricAlert](l.fixture, input.AlertKey)
	if !ok {
		return fmt.Errorf("metric_alert %q not found", input.AlertKey)
	}

	if input.CreatedAt.IsZero() {
		input.CreatedAt = alert.CreatedAt
	}
	if input.StateUpdatedAt.IsZero() {
		input.StateUpdatedAt = input.CreatedAt
	}
	if input.ValueTime.IsZero() {
		input.ValueTime = alert.CreatedAt
	}

	if input.Params.Query == "" {
		alertLoader := UnwrapLoader[*MetricAlertLoader](l.fixture, ModelMetricAlert)
		if alertInput, ok := alertLoader.GetInput(input.AlertKey); ok {
			if monitor, ok := Get[*models.MetricMonitor](l.fixture, alertInput.MonitorKey); ok {
				input.Params.Metrics = monitor.Params.Metrics
				input.Params.Query = monitor.Params.Query
				input.Params.Column = monitor.Params.Column
				input.Params.Resolution = monitor.Params.Resolution
			}
		}
	}

	if fake {
		populateBaseAlertEvent(l.fixture, &input.BaseAlertEvent)
	}
	return nil
}

func (l *MetricAlertEventLoader) PopulateModel(ctx context.Context, model *models.MetricAlertEvent, input *seedinput.MetricAlertEvent) ([]string, error) {
	populateBaseAlertEventModel(&model.BaseAlertEvent, &input.BaseAlertEvent)
	model.Params = input.Params
	return nil, nil // columns not used — custom Insert
}

func (l *MetricAlertEventLoader) Insert(ctx context.Context, model *models.MetricAlertEvent) (bool, error) {
	db := l.DB(ctx)

	alertID := model.AlertID

	// Find the alert by ID from loaded alerts.
	alertLoader := UnwrapLoader[*MetricAlertLoader](l.fixture, ModelMetricAlert)
	var baseAlert *models.BaseAlert
	for _, entry := range alertLoader.Loaded() {
		if entry.Value.ID == alertID {
			baseAlert = &entry.Value.BaseAlert
			break
		}
	}
	if baseAlert == nil {
		return false, fmt.Errorf("metric_alert with ID %d not found in loaded", alertID)
	}

	base, err := insertAlertEvent(ctx, db, baseAlert, model)
	if err != nil {
		return false, err
	}

	result := &models.MetricAlertEvent{BaseAlertEvent: *base}
	if err := base.Params.Decode(&result.Params); err != nil {
		return false, err
	}
	*model = *result

	// Re-select the parent alert so it picks up the new event relation.
	for key, entry := range alertLoader.Loaded() {
		if entry.Value.ID == alertID {
			if _, err := l.fixture.loaders[ModelMetricAlert].Select(ctx, key); err != nil {
				return false, err
			}
			break
		}
	}

	return true, nil
}

func (l *MetricAlertEventLoader) Update(ctx context.Context, model *models.MetricAlertEvent, columns []string) error {
	return sql.ErrNoRows
}

func (l *MetricAlertEventLoader) Select(ctx context.Context, key string) (*models.MetricAlertEvent, error) {
	if model, ok := l.Get(key); ok {
		return model, nil
	}
	return nil, fmt.Errorf("dbfixture: %s/%s not found in loaded", l.name, key)
}

// ---------------------------------------------------------------------------
// ErrorAlertEventLoader
// ---------------------------------------------------------------------------

// ErrorAlertEventLoader loads error alert event fixtures.
type ErrorAlertEventLoader struct {
	pgLoaderBase[seedinput.ErrorAlertEvent, models.ErrorAlertEvent]
}

// NewErrorAlertEventLoader creates a Loader for error alert events.
func NewErrorAlertEventLoader(f *Fixture, params LoaderParams) *ErrorAlertEventLoader {
	return &ErrorAlertEventLoader{pgLoaderBase: pgLoaderBase[seedinput.ErrorAlertEvent, models.ErrorAlertEvent]{
		params: params, fixture: f, name: ModelErrorAlertEvent,
	}}
}

func (l *ErrorAlertEventLoader) Resolve(ctx context.Context, input *seedinput.ErrorAlertEvent) error {
	alert, ok := Get[*models.ErrorAlert](l.fixture, input.AlertKey)
	if !ok {
		return fmt.Errorf("error_alert %q not found", input.AlertKey)
	}
	input.AlertID = alert.ID
	input.ProjectID = alert.ProjectID
	return nil
}

func (l *ErrorAlertEventLoader) Defaults(ctx context.Context, input *seedinput.ErrorAlertEvent, fake bool) error {
	alert, ok := Get[*models.ErrorAlert](l.fixture, input.AlertKey)
	if !ok {
		return fmt.Errorf("error_alert %q not found", input.AlertKey)
	}

	if input.CreatedAt.IsZero() {
		input.CreatedAt = alert.CreatedAt
	}
	if input.StateUpdatedAt.IsZero() {
		input.StateUpdatedAt = input.CreatedAt
	}
	if input.ValueTime.IsZero() {
		input.ValueTime = alert.CreatedAt
	}

	if input.Params.Query == "" {
		alertLoader := UnwrapLoader[*ErrorAlertLoader](l.fixture, ModelErrorAlert)
		if alertInput, ok := alertLoader.GetInput(input.AlertKey); ok {
			if monitor, ok := Get[*models.ErrorMonitor](l.fixture, alertInput.MonitorKey); ok {
				input.Params.Metrics = monitor.Params.Metrics
				input.Params.Query = monitor.Params.Query
			}
		}
	}

	if fake {
		populateBaseAlertEvent(l.fixture, &input.BaseAlertEvent)
	}
	return nil
}

func (l *ErrorAlertEventLoader) PopulateModel(ctx context.Context, model *models.ErrorAlertEvent, input *seedinput.ErrorAlertEvent) ([]string, error) {
	populateBaseAlertEventModel(&model.BaseAlertEvent, &input.BaseAlertEvent)
	model.Params = input.Params
	return nil, nil
}

func (l *ErrorAlertEventLoader) Insert(ctx context.Context, model *models.ErrorAlertEvent) (bool, error) {
	db := l.DB(ctx)
	alertID := model.AlertID

	var baseAlert *models.BaseAlert
	for _, entry := range UnwrapLoader[*ErrorAlertLoader](l.fixture, ModelErrorAlert).Loaded() {
		if entry.Value.ID == alertID {
			baseAlert = &entry.Value.BaseAlert
			break
		}
	}
	if baseAlert == nil {
		return false, fmt.Errorf("error_alert with ID %d not found in loaded", alertID)
	}

	base, err := insertAlertEvent(ctx, db, baseAlert, model)
	if err != nil {
		return false, err
	}

	result := &models.ErrorAlertEvent{BaseAlertEvent: *base}
	if err := base.Params.Decode(&result.Params); err != nil {
		return false, err
	}
	*model = *result

	alertLoader := UnwrapLoader[*ErrorAlertLoader](l.fixture, ModelErrorAlert)
	for key, m := range alertLoader.loaded {
		if m.Value.ID == alertID {
			if _, err := l.fixture.loaders[ModelErrorAlert].Select(ctx, key); err != nil {
				return false, err
			}
			break
		}
	}

	return true, nil
}

func (l *ErrorAlertEventLoader) Update(ctx context.Context, model *models.ErrorAlertEvent, columns []string) error {
	return sql.ErrNoRows
}

func (l *ErrorAlertEventLoader) Select(ctx context.Context, key string) (*models.ErrorAlertEvent, error) {
	if model, ok := l.Get(key); ok {
		return model, nil
	}
	return nil, fmt.Errorf("dbfixture: %s/%s not found in loaded", l.name, key)
}

// ---------------------------------------------------------------------------
// AlertNoteLoader
// ---------------------------------------------------------------------------

// AlertNoteLoader loads alert note fixtures.
type AlertNoteLoader struct {
	pgLoaderBase[seedinput.AlertNote, models.NoteAlertEvent]
}

// NewAlertNoteLoader creates a Loader for alert notes.
func NewAlertNoteLoader(f *Fixture, params LoaderParams) *AlertNoteLoader {
	return &AlertNoteLoader{pgLoaderBase: pgLoaderBase[seedinput.AlertNote, models.NoteAlertEvent]{
		params: params, fixture: f, name: ModelAlertNote,
	}}
}

func (l *AlertNoteLoader) Resolve(ctx context.Context, input *seedinput.AlertNote) error {
	if err := resolveProjectID(l.fixture, &input.ProjectID, input.ProjectKey); err != nil {
		return err
	}

	user, ok := Get[*models.User](l.fixture, input.UserKey)
	if !ok {
		return fmt.Errorf("user %q not found", input.UserKey)
	}
	input.UserID = user.ID

	if alert, ok := Get[*models.MetricAlert](l.fixture, input.AlertKey); ok {
		input.AlertID = alert.ID
	} else if alert, ok := Get[*models.ErrorAlert](l.fixture, input.AlertKey); ok {
		input.AlertID = alert.ID
	} else {
		return fmt.Errorf("alert %q not found in metric_alert or error_alert", input.AlertKey)
	}
	return nil
}

func (l *AlertNoteLoader) Defaults(ctx context.Context, input *seedinput.AlertNote, fake bool) error {
	if input.CreatedAt.IsZero() {
		input.CreatedAt = l.fixture.clock.Now()
	}
	if fake {
		if input.Text == nil || *input.Text == "" {
			text := gofakeit.LoremIpsumSentence(10)
			input.Text = &text
		}
	}
	return nil
}

func (l *AlertNoteLoader) PopulateModel(ctx context.Context, model *models.NoteAlertEvent, input *seedinput.AlertNote) ([]string, error) {
	// AlertNote uses a custom Insert that builds the event from input.
	// PopulateModel stores the essential fields on the model for Insert to use.
	model.AlertID = input.AlertID
	model.ProjectID = input.ProjectID
	model.UserID = input.UserID
	if !input.CreatedAt.IsZero() {
		model.CreatedAt = input.CreatedAt
	}
	if input.Text != nil {
		model.Params.Text = *input.Text
	}
	return nil, nil
}

func (l *AlertNoteLoader) Insert(ctx context.Context, model *models.NoteAlertEvent) (bool, error) {
	db := l.DB(ctx)

	// Build the event from model fields.
	base := &models.BaseAlertEvent{
		AlertID:   model.AlertID,
		ProjectID: model.ProjectID,
		UserID:    model.UserID,
	}
	event := models.NewNoteAlertEvent(base, model.Params)
	if !model.CreatedAt.IsZero() {
		event.CreatedAt = model.CreatedAt
	}
	if event.Status == "" {
		event.Status = models.AlertStatusUnresolved
	}
	if event.ActivityState == "" {
		event.ActivityState = models.AlertActivityNew
	}
	if event.Priority == "" {
		event.Priority = models.AlertPriorityLow
	}

	if _, err := db.NewInsert().Model(event).Exec(ctx); err != nil {
		return false, err
	}

	// Select back the inserted event.
	result := new(models.NoteAlertEvent)
	if err := db.NewSelect().Model(result).
		Where("alert_id = ?", event.AlertID).
		Where("user_id = ?", event.UserID).
		OrderExpr("id DESC").
		Limit(1).
		Scan(ctx); err != nil {
		return false, err
	}

	*model = *result
	return true, nil
}

// ---------------------------------------------------------------------------
// AlertAssignmentLoader
// ---------------------------------------------------------------------------

// AlertAssignmentLoader loads alert assignment fixtures.
type AlertAssignmentLoader struct {
	pgLoaderBase[seedinput.AlertAssignment, models.AlertAssignment]
}

// NewAlertAssignmentLoader creates a Loader for alert assignments.
func NewAlertAssignmentLoader(f *Fixture, params LoaderParams) *AlertAssignmentLoader {
	return &AlertAssignmentLoader{pgLoaderBase: pgLoaderBase[seedinput.AlertAssignment, models.AlertAssignment]{
		params: params, fixture: f, name: ModelAlertAssignment,
	}}
}

func (l *AlertAssignmentLoader) Resolve(ctx context.Context, input *seedinput.AlertAssignment) error {
	if alert, ok := Get[*models.MetricAlert](l.fixture, input.AlertKey); ok {
		input.AlertID = alert.ID
	} else if alert, ok := Get[*models.ErrorAlert](l.fixture, input.AlertKey); ok {
		input.AlertID = alert.ID
	} else {
		return fmt.Errorf("alert %q not found", input.AlertKey)
	}

	user, ok := Get[*models.User](l.fixture, input.UserKey)
	if !ok {
		return fmt.Errorf("user %q not found", input.UserKey)
	}
	input.UserID = user.ID
	return nil
}

func (l *AlertAssignmentLoader) PopulateModel(ctx context.Context, model *models.AlertAssignment, input *seedinput.AlertAssignment) ([]string, error) {
	model.AlertID = input.AlertID
	model.UserID = input.UserID
	return []string{"alert_id", "user_id"}, nil
}

func (l *AlertAssignmentLoader) Insert(ctx context.Context, model *models.AlertAssignment) (bool, error) {
	db := l.DB(ctx)
	res, err := db.NewInsert().Model(model).On("CONFLICT DO NOTHING").Exec(ctx)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()

	if err := db.NewSelect().Model(model).
		Where("alert_id = ?", model.AlertID).
		Where("user_id = ?", model.UserID).
		Scan(ctx); err != nil {
		return false, err
	}
	return n != 0, nil
}

func (l *AlertAssignmentLoader) ModelPK(model *models.AlertAssignment) map[string]any {
	return map[string]any{"alert_id": model.AlertID, "user_id": model.UserID}
}

func (l *AlertAssignmentLoader) Update(ctx context.Context, model *models.AlertAssignment, columns []string) error {
	return nil // all columns are PK — nothing to update
}

// ---------------------------------------------------------------------------
// Alert helpers
// ---------------------------------------------------------------------------

func initBaseAlert(f *Fixture, monitor models.Monitor, alert *seedinput.BaseAlert, fake bool) error {
	baseMonitor := monitor.Base()
	alert.MonitorID = baseMonitor.ID
	alert.ProjectID = baseMonitor.ProjectID

	if fake {
		if alert.Name == "" {
			alert.Name = gofakeit.Name()
		}
		if len(alert.Attrs) == 0 {
			numAttrs := gofakeit.Number(1, 10)
			if alert.Attrs == nil {
				alert.Attrs = make(map[string]any, numAttrs)
			}
			for range numAttrs {
				alert.Attrs[gofakeit.LoremIpsumWord()] = gofakeit.LoremIpsumWord()
			}
		}
	}

	hashSeed := monitor.CalcHash()
	attrs := mql.NewAttrsFrom(alert.Attrs)
	buf := attrs.Bytes(nil, nil)
	alert.AttrsHash = xxh3.HashSeed(buf, hashSeed)

	if alert.TrendAggFunc == "" {
		if fake {
			funcs := []models.TrendAggFuncName{
				models.TrendAggSum, models.TrendAggAvg, models.TrendAggMedian, models.TrendAggLast,
			}
			alert.TrendAggFunc = funcs[rand.IntN(len(funcs))]
		} else {
			alert.TrendAggFunc = baseMonitor.TrendAggFunc
		}
	}
	if alert.TrendUpdatedAt.IsZero() {
		alert.TrendUpdatedAt = f.clock.Now().Add(-time.Hour).Add(-time.Minute)
	}
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = f.clock.Now()
	}
	if alert.UpdatedAt.IsZero() {
		alert.UpdatedAt = alert.CreatedAt
	}
	return nil
}

// populateBaseAlertModel maps seedinput.BaseAlert fields to models.BaseAlert.
func populateBaseAlertModel(model *models.BaseAlert, input *seedinput.BaseAlert) []string {
	var cols []string
	if input.ProjectID != 0 && model.ProjectID != input.ProjectID {
		model.ProjectID = input.ProjectID
		cols = append(cols, "project_id")
	}
	if input.MonitorID != 0 && model.MonitorID != input.MonitorID {
		model.MonitorID = input.MonitorID
		cols = append(cols, "monitor_id")
	}
	if input.Name != "" && model.Name != input.Name {
		model.Name = input.Name
		cols = append(cols, "name")
	}
	if input.Type != "" && model.Type != input.Type {
		model.Type = input.Type
		cols = append(cols, "type")
	}
	if len(input.Attrs) > 0 {
		model.Attrs = models.NewTypedAttrsFrom(input.Attrs)
		cols = append(cols, "attrs")
	}
	if input.AttrsHash != 0 && model.AttrsHash != input.AttrsHash {
		model.AttrsHash = input.AttrsHash
		cols = append(cols, "attrs_hash")
	}
	if input.HourlyTrend != nil {
		model.HourlyTrend = input.HourlyTrend
		cols = append(cols, "hourly_trend")
	}
	if !input.TrendUpdatedAt.IsZero() && model.TrendUpdatedAt != input.TrendUpdatedAt {
		model.TrendUpdatedAt = input.TrendUpdatedAt
		cols = append(cols, "trend_updated_at")
	}
	if input.TrendAggFunc != "" && model.TrendAggFunc != input.TrendAggFunc {
		model.TrendAggFunc = input.TrendAggFunc
		cols = append(cols, "trend_agg_func")
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

func populateBaseAlertEvent(f *Fixture, event *seedinput.BaseAlertEvent) {
	if event.Name == "" {
		names := []models.AlertEventName{
			models.AlertEventCreated, models.AlertEventStatusChanged, models.AlertEventRecurring,
		}
		event.Name = names[rand.IntN(len(names))]
	}
	if event.Status == "" {
		statuses := []models.AlertStatus{
			models.AlertStatusUnresolved, models.AlertStatusResolved, models.AlertStatusArchived,
		}
		event.Status = statuses[rand.IntN(len(statuses))]
	}
	if event.ActivityState == "" {
		states := []models.AlertActivityState{
			models.AlertActivityNew, models.AlertActivityOngoing,
			models.AlertActivityEscalated, models.AlertActivityRegressed,
		}
		event.ActivityState = states[rand.IntN(len(states))]
	}
	if event.Priority == "" {
		priorities := []models.AlertPriority{
			models.AlertPriorityInfo, models.AlertPriorityLow,
			models.AlertPriorityMedium, models.AlertPriorityHigh,
		}
		event.Priority = priorities[rand.IntN(len(priorities))]
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = f.clock.Now()
	}
	if event.StateUpdatedAt.IsZero() {
		event.StateUpdatedAt = f.clock.Now()
	}
}

// populateBaseAlertEventModel maps seedinput.BaseAlertEvent fields to models.BaseAlertEvent.
func populateBaseAlertEventModel(model *models.BaseAlertEvent, input *seedinput.BaseAlertEvent) {
	model.AlertID = input.AlertID
	model.ProjectID = input.ProjectID
	model.Name = input.Name
	model.SerialNumber = input.SerialNumber
	model.Status = input.Status
	model.Priority = input.Priority
	model.ActivityState = input.ActivityState
	model.StateUpdatedAt = input.StateUpdatedAt
	model.ArchivedUntil = input.ArchivedUntil
	model.CurrentValue = input.CurrentValue
	model.ValueTime = input.ValueTime
	model.CreatedAt = input.CreatedAt
}

// alertEventInserter is implemented by model types that can be inserted as alert events.
type alertEventInserter interface {
	Base() *models.BaseAlertEvent
}

// insertAlertEvent inserts an alert event and updates the alert's event_id.
func insertAlertEvent(
	ctx context.Context, db bun.IDB,
	alert *models.BaseAlert, event alertEventInserter,
) (*models.BaseAlertEvent, error) {
	base := event.Base()
	base.AlertID = alert.ID
	base.ProjectID = alert.ProjectID
	if _, err := db.NewInsert().Model(event).Exec(ctx); err != nil {
		return nil, err
	}

	model := new(models.BaseAlertEvent)
	if err := db.NewSelect().
		Model(model).
		Where("alert_id = ?", alert.ID).
		Where("project_id = ?", alert.ProjectID).
		OrderExpr("id DESC").
		Limit(1).
		Scan(ctx); err != nil {
		return nil, err
	}

	alert.EventID = model.ID
	if _, err := db.NewUpdate().
		Model(alert).
		Set("event_id = ?", alert.EventID).
		Set("updated_at = ?", model.CreatedAt).
		Where("id = ?", alert.ID).
		Exec(ctx); err != nil {
		return nil, err
	}
	alert.Event = model

	return model, nil
}
