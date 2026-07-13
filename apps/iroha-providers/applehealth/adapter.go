// Package applehealth adapts Apple Health exports to the core provider
// observation contracts.
package applehealth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
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
		Capabilities: []provider.Capability{
			provider.CapabilityActivities,
			provider.CapabilitySleep,
			provider.CapabilityDailySummary,
			provider.CapabilityDailyMetrics,
		},
	}
}

func (a Adapter) ImportActivities(ctx context.Context, source provider.Source, options provider.ImportOptions) ([]observations.Activity, error) {
	if err := provider.ValidateRequested(a, options); err != nil {
		return nil, err
	}
	path, cleanup, err := materialize(ctx, source, "apple-health-*.zip")
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
	if err := provider.ValidateRequested(a, options); err != nil {
		return nil, err
	}
	path, cleanup, err := materialize(ctx, source, "apple-health-*.zip")
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
	if err := provider.ValidateRequested(a, options); err != nil {
		return provider.DailyObservations{}, err
	}
	path, cleanup, err := materialize(ctx, source, "apple-health-*.zip")
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

func materialize(ctx context.Context, source provider.Source, pattern string) (string, func(), error) {
	if source.Open == nil {
		return "", func() {}, adaptError(source, "open_source", errors.New("source opener is required"))
	}
	reader, err := source.Open(ctx)
	if err != nil {
		return "", func() {}, adaptError(source, "open_source", err)
	}
	temp, err := os.CreateTemp("", pattern)
	if err != nil {
		_ = reader.Close()
		return "", func() {}, adaptError(source, "create_temp_source", err)
	}
	path := temp.Name()
	cleanup := func() { _ = os.Remove(path) }
	_, copyErr := io.Copy(temp, contextReader{ctx: ctx, reader: reader})
	closeReaderErr := reader.Close()
	closeTempErr := temp.Close()
	if copyErr != nil {
		cleanup()
		return "", func() {}, adaptError(source, "copy_source", copyErr)
	}
	if closeReaderErr != nil {
		cleanup()
		return "", func() {}, adaptError(source, "close_source", closeReaderErr)
	}
	if closeTempErr != nil {
		cleanup()
		return "", func() {}, adaptError(source, "close_temp_source", closeTempErr)
	}
	return path, cleanup, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
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
