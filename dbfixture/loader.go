package dbfixture

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/brianvoe/gofakeit/v5"
	"github.com/goccy/go-yaml"
	"github.com/uptrace/bun"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/models"
	"github.com/uptrace/uptrace/pkg/unixtime"
	"github.com/uptrace/uptrace/pkg/validerr"
)

// ParseFile parses a fixture YAML file and returns a new fixture with seed data.
// templateData is passed to Go template rendering (sprig functions, etc.).
// Pass nil if the YAML file does not use templates.
func ParseFile(
	params FixtureParams, filename string, templateData any, opts ...Option,
) (*Fixture, *seedinput.SeedData, error) {
	f := NewFixture(params, opts...)

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}

	data, err = renderTemplate(data, templateData, f.clock)
	if err != nil {
		return nil, nil, err
	}

	seedData := new(seedinput.SeedData)
	if err := yaml.UnmarshalWithOptions(data, seedData, yaml.DisallowUnknownField()); err != nil {
		return nil, nil, fmt.Errorf("file: %s %w", filename, err)
	}

	return f, seedData, nil
}

// RunInTx runs fn inside a PostgreSQL transaction with the fixture context set,
// so that loader methods (Resolve, Insert, Update, Delete, Select, etc.) can
// cross-reference already-loaded models.
func (f *Fixture) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return f.pg.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		ctx = bunTxKey.WithValue(ctx, &tx)
		return fn(ctx)
	})
}

// Clear removes all loaded fixture data from the database.
func (f *Fixture) Clear(ctx context.Context) error {
	// Delete in reverse dependency order so foreign keys are respected.
	for i := len(f.order) - 1; i >= 0; i-- {
		name := f.order[i]
		loader, ok := f.Loader(name)
		if !ok {
			return fmt.Errorf("dbfixture: loader %q not registered", name)
		}

		// Iterate KeyStore so Clear also removes fixtures persisted from prior
		// runs (loaded into KeyStore via FixtureKeyRepo on startup), not just
		// what is currently in the loader's in-memory map.
		for key := range f.keys.Keys(name) {
			if err := loader.Delete(ctx, key); err != nil {
				return fmt.Errorf("clear %s/%s: %w", name, key, err)
			}
		}

		// Truncate pre-aggregated/group tables if the loader supports it.
		if u, ok := loader.(interface{ Unwrap() any }); ok {
			if clearer, ok := u.Unwrap().(AfterClearer); ok {
				if err := clearer.AfterClear(ctx); err != nil {
					return fmt.Errorf("after-clear %s: %w", name, err)
				}
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

// resolveProjectID resolves ProjectID from the override project or by key lookup.
// It is a no-op when projectID is already set.
func resolveProjectID(f *Fixture, projectID *uint32, projectKey string) error {
	if *projectID != 0 {
		return nil
	}
	if p := f.overrideProject(); p != nil {
		*projectID = p.ID
		return nil
	}
	project, ok := Get[*models.Project](f, projectKey)
	if !ok {
		return fmt.Errorf("project %q not found", projectKey)
	}
	*projectID = project.ID
	return nil
}

func resolveProject(f *Fixture, projectKey string) (*models.Project, error) {
	if p := f.overrideProject(); p != nil {
		return p, nil
	}
	project, ok := Get[*models.Project](f, projectKey)
	if !ok {
		return nil, fmt.Errorf("project %q not found", projectKey)
	}
	return project, nil
}

// ---------------------------------------------------------------------------
// YAML template rendering
// ---------------------------------------------------------------------------

func renderTemplate(
	data []byte, templateData any, clock unixtime.Clock,
) ([]byte, error) {
	tpl, err := template.New("").
		Funcs(sprig.TxtFuncMap()).
		Funcs(templateFuncs(clock)).
		Parse(string(data))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, templateData); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func templateFuncs(clock unixtime.Clock) template.FuncMap {
	return template.FuncMap{
		"now": func() unixtime.Nano {
			return clock.Now()
		},
		"interval": func(n int, durationStr string) []unixtime.Nano {
			duration, err := time.ParseDuration(durationStr)
			if err != nil {
				panic(validerr.Prefix("duration", err))
			}

			now := clock.Now()
			times := make([]unixtime.Nano, n)
			for i := range n {
				times[i] = unixtime.Nano(now.Add(time.Duration(i) * duration).UnixNano())
			}
			return times
		},
		"addDuration": func(now unixtime.Nano, durationStr string) unixtime.Nano {
			duration, err := time.ParseDuration(durationStr)
			if err != nil {
				panic(validerr.Prefix("duration", err))
			}

			return now.Add(duration)
		},
		"seq": func(n int) []int {
			result := make([]int, n)
			for i := range n {
				result[i] = i
			}
			return result
		},
		"randInt":   gofakeit.Number,
		"randFloat": gofakeit.Float64Range,
	}
}

