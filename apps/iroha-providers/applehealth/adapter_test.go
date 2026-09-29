package applehealth

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
)

func TestAdapterDescriptor(t *testing.T) {
	descriptor := New().Descriptor()
	if descriptor.ID != ProviderID || descriptor.AdapterVersion == "" {
		t.Fatalf("unexpected descriptor: %#v", descriptor)
	}
	if err := provider.ValidateAdapter(New()); err != nil {
		t.Fatalf("ValidateAdapter() error = %v", err)
	}
}

func TestAdapterRejectsSourceWithoutOpener(t *testing.T) {
	_, err := New().ImportAll(context.Background(), provider.Source{Kind: "zip"}, provider.ImportOptions{})
	if err == nil {
		t.Fatal("ImportAll() accepted source without opener")
	}
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) || providerErr.Kind != provider.ErrorInvalidSource {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestImportAllOpensSourceOnce(t *testing.T) {
	archivePath := writeMinimalExport(t)
	openCount := 0
	_, err := New().ImportAll(context.Background(), provider.Source{
		Kind:   "apple_health_export",
		SHA256: "test-digest",
		Open: func(context.Context) (io.ReadCloser, error) {
			openCount++
			return os.Open(archivePath)
		},
	}, provider.ImportOptions{})
	if err != nil {
		t.Fatalf("ImportAll() error = %v", err)
	}
	if openCount != 1 {
		t.Fatalf("source opened %d times, want 1", openCount)
	}
}

func writeMinimalExport(t *testing.T) string {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "apple-health-*.zip")
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	entry, err := archive.Create("export.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(`<HealthData></HealthData>`)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return file.Name()
}

func TestFullExportCoverage(t *testing.T) {
	loc := time.FixedZone("JST", 9*3600)
	if got := fullExportCoverage(provider.ImportBatch{}, loc); got != nil {
		t.Fatalf("empty batch asserted coverage: %+v", got)
	}
	batch := provider.ImportBatch{Daily: provider.DailyObservations{Metrics: []observations.DailyMetric{
		{Day: time.Date(2026, 9, 3, 0, 0, 0, 0, loc)},
		{Day: time.Date(2026, 9, 1, 0, 0, 0, 0, loc)},
	}}}
	got := fullExportCoverage(batch, loc)
	if len(got) != 1 {
		t.Fatalf("got %d assertions", len(got))
	}
	a := got[0]
	if a.Completeness != "covered" || a.IngestionMode != "full_snapshot" || a.Category != "health" {
		t.Fatalf("unexpected assertion %+v", a)
	}
	if !a.From.Equal(time.Date(2026, 9, 1, 0, 0, 0, 0, loc)) || !a.To.Equal(time.Date(2026, 9, 4, 0, 0, 0, 0, loc)) {
		t.Fatalf("window = %v..%v", a.From, a.To)
	}
}
