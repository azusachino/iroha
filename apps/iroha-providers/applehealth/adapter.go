// Package applehealth adapts Apple Health exports to the core provider
// observation contracts.
package applehealth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	"github.com/azusachino/iroha/apps/iroha-providers/internal/materialize"
	"github.com/azusachino/iroha/apps/iroha-providers/parsers"
)

const ProviderID = "apple_health"

type Adapter struct{}

func New() Adapter { return Adapter{} }

func (Adapter) Descriptor() provider.Descriptor {
	return provider.Descriptor{
		ID:             ProviderID,
		DisplayName:    "Apple Health",
		AdapterVersion: coreimports.DefaultParserVersion,
		Domains:        []provider.Domain{provider.DomainHealth},
		SourceKinds:    []string{coreimports.KindAppleHealthExport, coreimports.KindHealthAutoExport},
		Capabilities: []provider.Capability{
			provider.CapabilityHealthActivities,
			provider.CapabilityHealthSleep,
			provider.CapabilityHealthDailySummary,
			provider.CapabilityHealthDailyMetrics,
		},
	}
}

func (a Adapter) ImportAll(ctx context.Context, source provider.Source, options provider.ImportOptions) (provider.ImportBatch, error) {
	if source.Kind == coreimports.KindHealthAutoExport {
		path, cleanup, err := materialize.Source(ctx, source, ProviderID, "health-auto-export-*.json")
		if err != nil {
			return provider.ImportBatch{}, adaptError(source, "materialize_hae", err)
		}
		defer cleanup()
		loc, err := time.LoadLocation(options.Timezone)
		if err != nil {
			return provider.ImportBatch{}, adaptError(source, "load_timezone", err)
		}
		batch, err := parsers.ParseHealthAutoExport(path, source.SHA256, loc)
		if err != nil {
			return provider.ImportBatch{}, adaptError(source, "parse_hae", err)
		}
		return batch, nil
	}
	path, cleanup, err := materialize.Source(ctx, source, ProviderID, "apple-health-*.zip")
	if err != nil {
		return provider.ImportBatch{}, err
	}
	defer cleanup()
	activities, err := parsers.ParseAppleHealthExport(path, source.SHA256)
	if err != nil {
		return provider.ImportBatch{}, adaptError(source, "parse_activities", err)
	}
	sleep, err := parsers.ParseAppleHealthSleep(path)
	if err != nil {
		return provider.ImportBatch{}, adaptError(source, "parse_sleep", err)
	}
	summaries, metrics, err := parsers.ParseAppleHealthDailyActivity(path)
	if err != nil {
		return provider.ImportBatch{}, adaptError(source, "parse_daily", err)
	}
	batch := provider.ImportBatch{Activities: activities, Sleep: sleep, Daily: provider.DailyObservations{Summaries: summaries, Metrics: metrics}}
	loc, err := time.LoadLocation(options.Timezone)
	if err != nil {
		return provider.ImportBatch{}, adaptError(source, "load_timezone", err)
	}
	batch.Coverage = fullExportCoverage(batch, loc)
	return batch, nil
}

// fullExportCoverage asserts a full export covered the calendar days it
// contains. An export with no dated records asserts nothing.
func fullExportCoverage(batch provider.ImportBatch, loc *time.Location) []provider.CoverageAssertion {
	var first, last time.Time
	mark := func(t time.Time) {
		if t.IsZero() {
			return
		}
		if first.IsZero() || t.Before(first) {
			first = t
		}
		if t.After(last) {
			last = t
		}
	}
	for _, a := range batch.Activities {
		mark(a.StartedAt)
		if a.EndedAt != nil {
			mark(*a.EndedAt)
		}
	}
	for _, sl := range batch.Sleep {
		mark(sl.StartedAt)
		mark(sl.EndedAt)
	}
	for _, d := range batch.Daily.Summaries {
		mark(d.Day)
	}
	for _, d := range batch.Daily.Metrics {
		mark(d.Day)
	}
	if first.IsZero() {
		return nil
	}
	from, to := first.In(loc), last.In(loc)
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, loc)
	to = time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	return []provider.CoverageAssertion{{
		Category:      "health",
		ScopeJSON:     json.RawMessage(`{}`),
		From:          from,
		To:            to,
		Timezone:      loc.String(),
		IngestionMode: "full_snapshot",
		Completeness:  "covered",
	}}
}

func adaptError(source provider.Source, operation string, err error) error {
	if err == nil {
		return nil
	}
	kind := provider.ErrorInternal
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if operation == "open_source" || operation == "copy_source" {
		kind = provider.ErrorInvalidSource
	}
	return &provider.Error{
		Kind:       kind,
		Provider:   ProviderID,
		SourceKind: source.Kind,
		Op:         operation,
		Err:        fmt.Errorf("%w", err),
	}
}
