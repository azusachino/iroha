package applehealth

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	"github.com/azusachino/iroha/apps/iroha-providers/internal/materialize"
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
	_, err := New().ImportActivities(context.Background(), provider.Source{Kind: "zip"}, provider.ImportOptions{Requested: []provider.Capability{provider.CapabilityHealthActivities}})
	if err == nil {
		t.Fatal("ImportActivities() accepted source without opener")
	}
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) || providerErr.Kind != provider.ErrorInvalidSource {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestContextReaderStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := materialize.NewContextReader(ctx, strings.NewReader("data")).Read(make([]byte, 4))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Read() error = %v", err)
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
