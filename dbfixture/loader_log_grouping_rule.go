package dbfixture

import (
	"context"

	"github.com/uptrace/uptrace/dbfixture/seedinput"
	"github.com/uptrace/uptrace/tracing"
	"github.com/uptrace/uptrace/tracing/logparser/grokmatch"
	"github.com/uptrace/uptrace/tracing/logparser/grokpat"
)

// LogGroupingRuleLoader loads log grouping rule fixtures.
type LogGroupingRuleLoader struct {
	pgLoaderBase[seedinput.LogGroupingRule, tracing.LogGroupingRule]
}

// NewLogGroupingRuleLoader creates a Loader for log grouping rules.
func NewLogGroupingRuleLoader(f *Fixture, params LoaderParams) *LogGroupingRuleLoader {
	return &LogGroupingRuleLoader{pgLoaderBase: pgLoaderBase[seedinput.LogGroupingRule, tracing.LogGroupingRule]{
		params: params, fixture: f, name: ModelLogGroupingRule,
	}}
}

// Resolve resolves cross-references from already-loaded fixtures.
func (l *LogGroupingRuleLoader) Resolve(ctx context.Context, input *seedinput.LogGroupingRule) error {
	if input.ProjectKey == "" && l.fixture.overrideProjectKey == "" {
		return nil
	}
	return resolveProjectID(l.fixture, &input.ProjectID, input.ProjectKey)
}

// PopulateModel maps fields from input onto model.
func (l *LogGroupingRuleLoader) PopulateModel(
	ctx context.Context, model *tracing.LogGroupingRule, input *seedinput.LogGroupingRule,
) ([]string, error) {
	var cols []string

	if input.ProjectID != 0 && model.ProjectID != input.ProjectID {
		model.ProjectID = input.ProjectID
		cols = append(cols, "project_id")
	}
	if len(input.Patterns) > 0 {
		model.Patterns = input.Patterns
		cols = append(cols, "patterns")
	}
	if input.Status != "" {
		status := tracing.LogGroupingRuleStatus(input.Status)
		if model.Status != status {
			model.Status = status
			cols = append(cols, "status")
		}
	}
	if input.Error != "" && model.Error != input.Error {
		model.Error = input.Error
		cols = append(cols, "error")
	}

	return cols, nil
}

// Validate validates the model after population.
func (l *LogGroupingRuleLoader) Validate(ctx context.Context, model *tracing.LogGroupingRule) error {
	// Non-active rules are inert — skip conflict checking.
	if model.Status != "" && model.Status != tracing.LogGroupingRuleStatusActive {
		return nil
	}

	// Build a matcher from all loaded rules except this one for conflict checking.
	m := grokmatch.New()

	for _, entry := range l.Loaded() {
		rule := entry.Value
		if rule.ID == model.ID {
			continue
		}
		// Ignore non-active rules when building the conflict check matcher.
		if rule.Status != "" && rule.Status != tracing.LogGroupingRuleStatusActive {
			continue
		}
		for _, patternStr := range rule.Patterns {
			pat, err := grokpat.New(patternStr)
			if err != nil {
				return err
			}
			if err := m.Add(pat, rule.ID); err != nil {
				return err
			}
		}
	}

	return model.Validate(m)
}

// KeyByID returns the fixture key for a rule ID.
func (l *LogGroupingRuleLoader) KeyByID(ruleID uint64) (string, bool) {
	for key, entry := range l.Loaded() {
		rule := entry.Value
		if rule.ID == ruleID {
			return key, true
		}
	}
	return "", false
}
