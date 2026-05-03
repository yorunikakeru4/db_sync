package dbfixture

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/brianvoe/gofakeit/v5"
	"github.com/uptrace/bun"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
)

// ---------------------------------------------------------------------------
// notifChannelLoaderBase — shared logic for all notification channel loaders
// ---------------------------------------------------------------------------

// notifChannelLoaderBase provides shared methods for notification channel loaders.
// Concrete channel types embed this struct.
type notifChannelLoaderBase[Input any, Model any] struct {
	pgLoaderBase[Input, Model]

	channelType  models.NotifChannelType
	baseFunc     func(input *Input) *seedinput.BaseNotifChannel
	decodeFunc   func(base *models.BaseNotifChannel) (*Model, error)
	newModelFunc func() *Model
}

func (l *notifChannelLoaderBase[Input, Model]) NewModel() *Model {
	if l.newModelFunc != nil {
		return l.newModelFunc()
	}
	return l.pgLoaderBase.NewModel()
}

func (l *notifChannelLoaderBase[Input, Model]) Resolve(ctx context.Context, input *Input) error {
	return resolveNotifChannel(l.fixture, l.baseFunc(input))
}

func (l *notifChannelLoaderBase[Input, Model]) Defaults(ctx context.Context, input *Input, fake bool) error {
	base := l.baseFunc(input)
	base.Type = l.channelType
	defaultsNotifChannel(l.fixture, base)
	if fake {
		fakeDataNotifChannel(l.fixture, base)
	}
	return nil
}

func (l *notifChannelLoaderBase[Input, Model]) PopulateModel(ctx context.Context, model *Model, input *Input) ([]string, error) {
	inputBase := l.baseFunc(input)
	modelBase := l.getBaseFromModel(model)
	if modelBase == nil {
		return nil, fmt.Errorf("notif channel model has no base")
	}

	var cols []string
	if inputBase.ProjectID != 0 && modelBase.ProjectID != inputBase.ProjectID {
		modelBase.ProjectID = inputBase.ProjectID
		cols = append(cols, "project_id")
	}
	if inputBase.Name != nil && modelBase.Name != *inputBase.Name {
		modelBase.Name = *inputBase.Name
		cols = append(cols, "name")
	}
	if inputBase.Status != nil && modelBase.Status != *inputBase.Status {
		modelBase.Status = *inputBase.Status
		cols = append(cols, "status")
	}
	modelBase.Type = l.channelType

	// Derive MatchAll from MonitorKeys: if no specific monitors are listed,
	// the channel matches all monitors. Computed here (not in Defaults) so
	// updates that change MonitorKeys also update MatchAll.
	matchAll := len(inputBase.MonitorKeys) == 0
	if inputBase.MatchAll != nil {
		matchAll = *inputBase.MatchAll
	}
	if modelBase.MatchAll != matchAll {
		modelBase.MatchAll = matchAll
		cols = append(cols, "match_all")
	}
	if inputBase.Priorities != nil {
		modelBase.Priorities = inputBase.Priorities
		cols = append(cols, "priorities")
	}
	if inputBase.Condition != nil && modelBase.Condition != *inputBase.Condition {
		modelBase.Condition = *inputBase.Condition
		cols = append(cols, "condition")
	}

	// Resolve monitor keys to IDs for channel-monitor links.
	if len(inputBase.MonitorKeys) > 0 {
		monitorIDs, err := resolveMonitorKeys(l.fixture, inputBase.MonitorKeys)
		if err != nil {
			return nil, err
		}
		modelBase.MonitorIDs = monitorIDs
	}

	return cols, nil
}

// Insert inserts the channel by inserting the input (which has proper bun tags),
// then selects back the decoded model. Notif channel models have a complex
// input→base→model decode path, so we insert the input directly.
//
// Note: this still uses the input for DB insert via the stored input reference.
// The model parameter is used to store the result.
func (l *notifChannelLoaderBase[Input, Model]) Insert(ctx context.Context, model *Model) (bool, error) {
	db := l.DB(ctx)

	base := l.getBaseFromModel(model)
	if base == nil {
		return false, fmt.Errorf("notif channel model has no base")
	}
	base.Type = l.channelType

	// Insert the model — bun serializes both inherited base fields and typed Params.
	if _, err := db.NewInsert().Model(model).Exec(ctx); err != nil {
		return false, err
	}

	// Select back by natural key to get the auto-generated ID.
	inserted := new(models.BaseNotifChannel)
	if err := db.NewSelect().Model(inserted).
		Where("c.type = ?", l.channelType).
		Where("c.project_id = ?", base.ProjectID).
		Where("c.name = ?", base.Name).
		OrderExpr("c.id DESC").
		Limit(1).
		Scan(ctx); err != nil {
		return false, err
	}

	if len(base.MonitorIDs) > 0 {
		if err := insertChannelMonitors(ctx, db, inserted.ID, base.MonitorIDs); err != nil {
			return false, err
		}
	}

	// Select back the full decoded model.
	full, err := l.selectNotifChannelByID(ctx, db, inserted.ID)
	if err != nil {
		return false, err
	}
	*model = *full
	return true, nil
}

// getBaseFromModel extracts the BaseNotifChannel from a model via interface.
func (l *notifChannelLoaderBase[Input, Model]) getBaseFromModel(model *Model) *models.BaseNotifChannel {
	type baseGetter interface {
		Base() *models.BaseNotifChannel
	}
	if bg, ok := any(model).(baseGetter); ok {
		return bg.Base()
	}
	return nil
}

func (l *notifChannelLoaderBase[Input, Model]) selectNotifChannelByID(
	ctx context.Context, db bun.IDB, id uint64,
) (*Model, error) {
	baseModel := new(models.BaseNotifChannel)
	if err := db.NewSelect().Model(baseModel).
		Where("c.id = ?", id).
		Scan(ctx); err != nil {
		return nil, err
	}
	return l.decodeFunc(baseModel)
}

func (l *notifChannelLoaderBase[Input, Model]) Select(ctx context.Context, key string) (*Model, error) {
	id, ok := l.pkID(key)
	if !ok {
		return nil, fmt.Errorf("dbfixture: %s/%s fixture key not found (not loaded or deleted)", l.name, key)
	}
	model, err := l.selectNotifChannelByID(ctx, l.DB(ctx), id)
	if err != nil {
		return nil, err
	}
	l.StoreLoaded(key, model, false)
	return model, nil
}

func (l *notifChannelLoaderBase[Input, Model]) Update(ctx context.Context, model *Model, columns []string) error {
	return l.pgLoaderBase.Update(ctx, model, columns)
}

// ---------------------------------------------------------------------------
// Concrete notification channel loaders — see loader_notif_channel_gen.go
// ---------------------------------------------------------------------------

//go:generate go run ./cmd/genloader

func newNotifChannelLoader[Input any, Model any](
	f *Fixture, params LoaderParams,
	name string, channelType models.NotifChannelType,
	baseFunc func(input *Input) *seedinput.BaseNotifChannel,
	decodeFunc func(base *models.BaseNotifChannel) (*Model, error),
	newModelFunc func() *Model,
) *notifChannelLoaderBase[Input, Model] {
	return &notifChannelLoaderBase[Input, Model]{
		pgLoaderBase: pgLoaderBase[Input, Model]{
			params: params, fixture: f, name: name,
		},
		channelType:  channelType,
		baseFunc:     baseFunc,
		decodeFunc:   decodeFunc,
		newModelFunc: newModelFunc,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func resolveNotifChannel(f *Fixture, base *seedinput.BaseNotifChannel) error {
	return resolveProjectID(f, &base.ProjectID, base.ProjectKey)
}

func defaultsNotifChannel(f *Fixture, base *seedinput.BaseNotifChannel) {
}

func fakeDataNotifChannel(f *Fixture, base *seedinput.BaseNotifChannel) {
	if base.Name == nil || *base.Name == "" {
		name := gofakeit.Name()
		base.Name = &name
	}
	if base.Status == nil || *base.Status == "" {
		statuses := []models.NotifChannelStatus{
			models.NotifChannelDraft, models.NotifChannelDelivering,
			models.NotifChannelPaused, models.NotifChannelDisabled,
		}
		status := statuses[rand.IntN(len(statuses))]
		base.Status = &status
	}
	if len(base.Priorities) == 0 {
		priorities := []models.AlertPriority{
			models.AlertPriorityInfo, models.AlertPriorityLow,
			models.AlertPriorityMedium, models.AlertPriorityHigh,
		}
		base.Priorities = []models.AlertPriority{priorities[rand.IntN(len(priorities))]}
	}
}

func resolveMonitorKeys(f *Fixture, keys []string) ([]uint64, error) {
	ids := make([]uint64, 0, len(keys))
	for _, key := range keys {
		if m, ok := Get[*models.MetricMonitor](f, key); ok {
			ids = append(ids, m.ID)
		} else if m, ok := Get[*models.ErrorMonitor](f, key); ok {
			ids = append(ids, m.ID)
		} else {
			return nil, fmt.Errorf("monitor %q not found", key)
		}
	}
	return ids, nil
}

func insertChannelMonitors(ctx context.Context, db bun.IDB, channelID uint64, monitorIDs []uint64) error {
	for _, monitorID := range monitorIDs {
		mc := &models.MonitorChannel{
			MonitorID: monitorID,
			ChannelID: channelID,
		}
		if _, err := db.NewInsert().Model(mc).On("CONFLICT DO NOTHING").Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
