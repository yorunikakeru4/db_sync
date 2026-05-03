package dbfixture

import (
	"context"
	"fmt"
	"strconv"

	"github.com/brianvoe/gofakeit/v5"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/attrval"
	"github.com/uptrace/uptrace/pkg/clickhouse/ch"
	"github.com/uptrace/uptrace/pkg/hashutil"
	"github.com/uptrace/uptrace/pkg/idgen"
	"github.com/uptrace/uptrace/pkg/spancol"
	"github.com/uptrace/uptrace/pkg/validerr"
	"github.com/uptrace/uptrace/tracing"
)

// ---------------------------------------------------------------------------
// SpanLoader
// ---------------------------------------------------------------------------

// SpanLoader handles spans keyed by SpanID, stored in ClickHouse.
type SpanLoader struct {
	chSpanLoaderBase
}

func NewSpanLoader(f *Fixture, params LoaderParams) *SpanLoader {
	return &SpanLoader{chSpanLoaderBase: chSpanLoaderBase{
		params:   params,
		fixture:  f,
		name:     ModelSpan,
		idTables: []string{tracing.TableSpansIndex, tracing.TableSpansData},
	}}
}

func (l *SpanLoader) NewInput() *seedinput.Span { return new(seedinput.Span) }

// Insert inserts the model into ClickHouse index and data tables.
func (l *SpanLoader) Insert(ctx context.Context, model *models.Span) (bool, error) {
	return l.insertSpanRecord(ctx, model,
		&tracing.SpanIndex{Span: *model}, new(tracing.SpanData),
		tracing.TableSpansIndex, tracing.TableSpansData)
}

// AfterClear truncates pre-aggregated span group tables.
func (l *SpanLoader) AfterClear(ctx context.Context) error {
	db := l.params.CHForWriting()
	for _, table := range []string{
		tracing.TableSpanLinks,
		tracing.TableSpanGroupMinutes.Name,
		tracing.TableSpanGroupHours.Name,
	} {
		if _, err := db.Exec(ctx, "TRUNCATE TABLE ?", ch.Name(table)); err != nil {
			return err
		}
	}
	return nil
}

// SpanIDKey converts a SpanID to a string key for type-erased access.
func SpanIDKey(id idgen.SpanID) string {
	return strconv.FormatUint(uint64(id), 10)
}

// InitSpanTraceIDs resolves trace IDs across all spans by following parent chains.
// Must be called before processing individual spans.
func InitSpanTraceIDs(spans map[idgen.SpanID]*seedinput.Span) error {
	for id := range spans {
		if _, err := populateTraceID(spans, id); err != nil {
			return err
		}
	}
	return nil
}

func populateTraceID(spans map[idgen.SpanID]*seedinput.Span, id idgen.SpanID) (idgen.TraceID, error) {
	span, ok := spans[id]
	if !ok {
		return idgen.TraceID{}, validerr.NotFound("span", id)
	}
	if span.ParentID == 0 {
		if span.TraceID.IsZero() {
			span.TraceID = span.NewTraceID()
		}
		return span.TraceID, nil
	}
	parentTraceID, err := populateTraceID(spans, span.ParentID)
	if err != nil {
		return idgen.TraceID{}, err
	}
	span.TraceID = parentTraceID
	return parentTraceID, nil
}

// spanDefaults sets default values on a span input.
func spanDefaults(
	f *Fixture,
	span *seedinput.Span,
	assignGroupID bool,
	hasher *hashutil.Hasher,
) error {
	if span.StartTime.IsZero() {
		span.StartTime = f.clock.Now()
	}

	if assignGroupID && span.GroupID == 0 {
		// Build a temporary model to compute the group hash.
		// ProjectID must be zero during hashing — GroupHash uses the project
		// parameter separately; a non-zero ProjectID would change the hash.
		model := spanFromInput(span, f)
		model.ProjectID = 0
		project, err := resolveProject(f, span.ProjectKey)
		if err != nil {
			return err
		}
		if hasher == nil {
			hasher = hashutil.NewHasher()
		}
		hasher.Reset()
		model.Hash = tracing.GroupHash(hasher, project, model)
		span.GroupID = model.Hash
	}

	return nil
}

// spanFromInput converts a seedinput.Span into a models.Span.
// f is used to resolve span links via the SpanLoader's loaded map; it may be nil.
func spanFromInput(span *seedinput.Span, f *Fixture) *models.Span {
	statusCode, statusMessage := spanStatusAttrs(span.Attrs)

	model := &models.Span{
		ID:            span.ID,
		ProjectID:     span.ProjectID,
		TraceID:       span.TraceID,
		ParentID:      span.ParentID,
		Type:          span.Type,
		System:        span.System,
		GroupID:       span.GroupID,
		CountX100:     span.CountX100,
		Name:          span.Name,
		DisplayName:   span.Name,
		Attrs:         models.NewTypedAttrsFrom(span.Attrs),
		Duration:      span.Duration,
		Size:          span.Size,
		StartTime:     span.StartTime,
		StatusCode:    statusCode,
		StatusMessage: statusMessage,
		TextIndex:     []byte(span.TextIndex),
		Links:         make([]*models.SpanLink, 0, len(span.Links)),
	}
	if f != nil {
		for _, link := range span.Links {
			if linkedSpan := f.GetSpan(link.SpanID); linkedSpan != nil {
				model.Links = append(model.Links, &models.SpanLink{
					SpanID:  linkedSpan.ID,
					TraceID: linkedSpan.TraceID,
					Attrs:   models.NewTypedAttrsFrom(link.Attrs),
				})
			}
		}
	}
	return model
}

// spanStatusAttrs extracts status fields from fixture attrs when present.
func spanStatusAttrs(attrs map[string]any) (statusCode, statusMessage string) {
	if attrs == nil {
		return "", ""
	}

	if value, ok := attrs[spancol.StatusCode]; ok {
		statusCode = attrString(value)
	}
	if value, ok := attrs[spancol.StatusMessage]; ok {
		statusMessage = attrString(value)
	}

	return statusCode, statusMessage
}

// attrString converts a fixture attr value into a string field value.
func attrString(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func populateSpan(f *Fixture, span *models.Span) {
	if span.Name == "" {
		span.Name = gofakeit.AppName()
	}
	if span.DisplayName == "" {
		span.DisplayName = gofakeit.AppName()
	}
	if span.EventName == "" {
		span.EventName = gofakeit.AppName()
	}
	if span.Kind == "" {
		span.Kind = gofakeit.RandomString(tracing.SpanKindEnum)
	}
	if span.Type == "" {
		span.Type = gofakeit.RandomString([]string{
			tracing.TypeSpanFuncs,
			tracing.TypeSpanHTTPServer,
			tracing.TypeSpanHTTPClient,
			tracing.TypeSpanDB,
			tracing.TypeSpanRPC,
			tracing.TypeSpanMessaging,
			tracing.TypeSpanFAAS,
			tracing.TypeSpanCLI,
			tracing.TypeLog,
			tracing.TypeEventsMessage,
			tracing.TypeEventsOther,
		})
	}
	if span.System == "" {
		switch span.Type {
		case "":
			span.System = gofakeit.RandomString([]string{
				tracing.SystemUnknown,
				tracing.SystemAll,
			})
		case tracing.TypeLog:
			span.System = gofakeit.RandomString([]string{
				tracing.SystemLogAll,
				tracing.SystemLogError,
				tracing.SystemLogFatal,
			})
		case tracing.TypeEventsMessage, tracing.TypeEventsOther:
			span.System = tracing.SystemEventsAll
		default:
			span.System = tracing.SystemSpansAll
		}
	}
	if span.StatusCode == "" {
		span.StatusCode = gofakeit.RandomString([]string{
			tracing.StatusCodeUnset,
			tracing.StatusCodeError,
			tracing.StatusCodeOK,
		})
	}
	if span.StatusMessage == "" {
		span.StatusMessage = gofakeit.HackerPhrase()
	}
	if span.Duration == 0 {
		span.Duration = models.Microseconds(gofakeit.Number(100, 10_000_000))
	}
	if span.Attrs.Len() == 0 {
		ln := gofakeit.Number(1, 10)
		span.Attrs = models.NewTypedAttrs(ln)
		for range ln {
			key := gofakeit.LoremIpsumWord()
			value := attrval.String(gofakeit.LoremIpsumWord())
			span.Attrs.Put(key, value)
		}
	}
	if span.StartTime.IsZero() {
		span.StartTime = f.clock.Now()
	}
}

func insertSpanLinks(params LoaderParams, ctx context.Context, project *models.Project, span *models.Span) error {
	if len(span.Links) == 0 {
		return nil
	}
	links := make([]*tracing.SpanBacklink, 0, len(span.Links))
	for _, link := range span.Links {
		links = append(links, &tracing.SpanBacklink{
			ProjectID:   project.ID,
			DestTraceID: span.TraceID,
			DestSpanID:  span.ID,
			SrcSpanID:   link.SpanID,
			SrcTraceID:  link.TraceID,
			Time:        span.StartTime,
			Attrs:       span.Attrs,
		})
	}
	db := params.CHForWriting()
	_, err := db.NewInsert().
		Model(&links).
		DistModelTable(tracing.TableSpanLinks).
		Exec(ctx)
	return err
}
