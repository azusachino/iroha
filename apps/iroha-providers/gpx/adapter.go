package gpx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
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
	path, cleanup, err := materialize(ctx, source)
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

func materialize(ctx context.Context, source provider.Source) (string, func(), error) {
	if source.Open == nil {
		return "", func() {}, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "open_source", Err: errors.New("source opener is required")}
	}
	reader, err := source.Open(ctx)
	if err != nil {
		return "", func() {}, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "open_source", Err: err}
	}
	temp, err := os.CreateTemp("", "iroha-gpx-*.gpx")
	if err != nil {
		_ = reader.Close()
		return "", func() {}, &provider.Error{Kind: provider.ErrorInternal, Provider: ProviderID, SourceKind: source.Kind, Op: "create_temp_source", Err: err}
	}
	path := temp.Name()
	cleanup := func() { _ = os.Remove(path) }
	_, copyErr := io.Copy(temp, contextReader{ctx: ctx, reader: reader})
	readerErr := reader.Close()
	tempErr := temp.Close()
	if copyErr != nil || readerErr != nil || tempErr != nil {
		cleanup()
		cause := copyErr
		if cause == nil {
			cause = readerErr
		}
		if cause == nil {
			cause = tempErr
		}
		return "", func() {}, &provider.Error{Kind: provider.ErrorInvalidSource, Provider: ProviderID, SourceKind: source.Kind, Op: "materialize_source", Err: fmt.Errorf("%w", cause)}
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
