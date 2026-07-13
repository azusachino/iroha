// Package applehealth adapts Apple Health exports to the core provider
// observation contracts.
package applehealth

import (
	"context"
	"errors"
	"fmt"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
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
		SourceKinds:    []string{coreimports.KindAppleHealthExport},
		Capabilities: []provider.Capability{
			provider.CapabilityHealthActivities,
			provider.CapabilityHealthSleep,
			provider.CapabilityHealthDailySummary,
			provider.CapabilityHealthDailyMetrics,
		},
	}
}

func (a Adapter) ImportAll(ctx context.Context, source provider.Source, options provider.ImportOptions) (provider.ImportBatch, error) {
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
	return provider.ImportBatch{Activities: activities, Sleep: sleep, Daily: provider.DailyObservations{Summaries: summaries, Metrics: metrics}}, nil
}

func (a Adapter) ImportActivities(ctx context.Context, source provider.Source, options provider.ImportOptions) ([]observations.Activity, error) {
	path, cleanup, err := materialize.Source(ctx, source, ProviderID, "apple-health-*.zip")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	items, err := parsers.ParseAppleHealthExport(path, source.SHA256)
	if err != nil {
		return nil, adaptError(source, "parse_activities", err)
	}
	return items, nil
}

func (a Adapter) ImportSleep(ctx context.Context, source provider.Source, options provider.ImportOptions) ([]observations.Sleep, error) {
	path, cleanup, err := materialize.Source(ctx, source, ProviderID, "apple-health-*.zip")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	items, err := parsers.ParseAppleHealthSleep(path)
	if err != nil {
		return nil, adaptError(source, "parse_sleep", err)
	}
	return items, nil
}

func (a Adapter) ImportDaily(ctx context.Context, source provider.Source, options provider.ImportOptions) (provider.DailyObservations, error) {
	path, cleanup, err := materialize.Source(ctx, source, ProviderID, "apple-health-*.zip")
	if err != nil {
		return provider.DailyObservations{}, err
	}
	defer cleanup()
	summaries, metrics, err := parsers.ParseAppleHealthDailyActivity(path)
	if err != nil {
		return provider.DailyObservations{}, adaptError(source, "parse_daily", err)
	}
	return provider.DailyObservations{Summaries: summaries, Metrics: metrics}, nil
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
