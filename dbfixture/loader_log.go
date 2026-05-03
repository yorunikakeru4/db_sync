package dbfixture

import (
	"context"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/clickhouse/ch"
	"github.com/uptrace/uptrace/tracing"
)

// ---------------------------------------------------------------------------
// LogLoader
// ---------------------------------------------------------------------------

// LogLoader loads log fixtures into ClickHouse.
type LogLoader struct {
	chSpanLoaderBase
}

// NewLogLoader creates a Loader for logs.
func NewLogLoader(f *Fixture, params LoaderParams) *LogLoader {
	return &LogLoader{chSpanLoaderBase: chSpanLoaderBase{
		params:   params,
		fixture:  f,
		name:     ModelLog,
		idTables: []string{tracing.TableLogsIndex, tracing.TableLogsData},
	}}
}

func (l *LogLoader) NewInput() *seedinput.Span { return new(seedinput.Span) }

func (l *LogLoader) Insert(ctx context.Context, model *models.Span) (bool, error) {
	return l.insertSpanRecord(ctx, model,
		&tracing.LogIndex{Span: *model}, new(tracing.LogData),
		tracing.TableLogsIndex, tracing.TableLogsData)
}

// AfterClear truncates pre-aggregated log group tables.
func (l *LogLoader) AfterClear(ctx context.Context) error {
	db := l.params.CHForWriting()
	for _, table := range []string{
		tracing.TableLogGroupMinutes.Name,
		tracing.TableLogGroupHours.Name,
	} {
		if _, err := db.Exec(ctx, "TRUNCATE TABLE ?", ch.Name(table)); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// EventLoader
// ---------------------------------------------------------------------------

// EventLoader loads event fixtures into ClickHouse.
type EventLoader struct {
	chSpanLoaderBase
}

// NewEventLoader creates a Loader for events.
func NewEventLoader(f *Fixture, params LoaderParams) *EventLoader {
	return &EventLoader{chSpanLoaderBase: chSpanLoaderBase{
		params:   params,
		fixture:  f,
		name:     ModelEvent,
		idTables: []string{tracing.TableEventsIndex, tracing.TableEventsData},
	}}
}

func (l *EventLoader) NewInput() *seedinput.Span { return new(seedinput.Span) }

func (l *EventLoader) Insert(ctx context.Context, model *models.Span) (bool, error) {
	return l.insertSpanRecord(ctx, model,
		&tracing.EventIndex{Span: *model}, new(tracing.EventData),
		tracing.TableEventsIndex, tracing.TableEventsData)
}

// AfterClear truncates pre-aggregated event group tables.
func (l *EventLoader) AfterClear(ctx context.Context) error {
	db := l.params.CHForWriting()
	for _, table := range []string{
		tracing.TableEventGroupMinutes.Name,
		tracing.TableEventGroupHours.Name,
	} {
		if _, err := db.Exec(ctx, "TRUNCATE TABLE ?", ch.Name(table)); err != nil {
			return err
		}
	}
	return nil
}
