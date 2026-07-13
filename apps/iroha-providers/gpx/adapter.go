package gpx

import (
	"context"
	"path/filepath"

	"github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	"github.com/azusachino/iroha/apps/iroha-providers/internal/materialize"
	"github.com/azusachino/iroha/apps/iroha-providers/parsers"
)

const ProviderID = "gpx"

type Adapter struct{}

func New() Adapter { return Adapter{} }

func (Adapter) Descriptor() provider.Descriptor {
	return provider.Descriptor{
		ID:             ProviderID,
		DisplayName:    "GPX",
		AdapterVersion: "gpx-2026-07-1",
		Domains:        []provider.Domain{provider.DomainHealth},
		SourceKinds:    []string{imports.KindGPX},
		Capabilities:   []provider.Capability{provider.CapabilityHealthActivities},
	}
}

func (a Adapter) ImportActivities(ctx context.Context, source provider.Source, options provider.ImportOptions) ([]observations.Activity, error) {
	if err := provider.ValidateRequested(a, options); err != nil {
		return nil, err
	}
	path, cleanup, err := materialize.Source(ctx, source, ProviderID, "iroha-gpx-*.gpx")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	title := filepath.Base(source.OriginalFilename)
	activities, err := parsers.ParseGPXFile(path, parsers.GPXOptions{Title: title, ExternalID: source.SHA256})
	if err != nil {
		return nil, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "parse_activities", Err: err}
	}
	return activities, nil
}
