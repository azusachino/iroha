package v1alpha1

import (
	"context"
	"io"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

type fakeAdapter struct {
	descriptor Descriptor
}

func (f fakeAdapter) Descriptor() Descriptor { return f.descriptor }

type fakeHealthAdapter struct{ fakeAdapter }

func (fakeHealthAdapter) ImportActivities(context.Context, Source, ImportOptions) ([]observations.Activity, error) {
	return nil, nil
}

func (fakeHealthAdapter) ImportSleep(context.Context, Source, ImportOptions) ([]observations.Sleep, error) {
	return nil, nil
}

func (fakeHealthAdapter) ImportDaily(context.Context, Source, ImportOptions) (DailyObservations, error) {
	return DailyObservations{}, nil
}

func TestNewRegistryValidatesDeclaredCapabilities(t *testing.T) {
	adapter := fakeHealthAdapter{fakeAdapter{descriptor: Descriptor{
		ID:             "apple_health",
		AdapterVersion: "test-v1",
		Capabilities:   []Capability{CapabilityActivities, CapabilitySleep, CapabilityDailyMetrics},
	}}}

	registry, err := NewRegistry(adapter)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	if got, ok := registry.Get("apple_health"); !ok || got == nil {
		t.Fatal("registry did not return registered adapter")
	}
	if got := registry.List(); len(got) != 1 || got[0].ID != "apple_health" {
		t.Fatalf("registry.List() = %#v", got)
	}
}

func TestNewRegistryRejectsCapabilityWithoutInterface(t *testing.T) {
	adapter := fakeAdapter{descriptor: Descriptor{
		ID:             "garmin",
		AdapterVersion: "test-v1",
		Capabilities:   []Capability{CapabilityActivities},
	}}

	if _, err := NewRegistry(adapter); err == nil {
		t.Fatal("NewRegistry() accepted an unimplemented capability")
	}
}

func TestValidateRequestedRejectsUnsupportedCapability(t *testing.T) {
	adapter := fakeHealthAdapter{fakeAdapter{descriptor: Descriptor{
		ID:             "apple_health",
		AdapterVersion: "test-v1",
		Capabilities:   []Capability{CapabilityActivities},
	}}}

	err := ValidateRequested(adapter, ImportOptions{Requested: []Capability{CapabilitySleep}})
	if err == nil {
		t.Fatal("ValidateRequested() accepted an unsupported capability")
	}
}

func TestSourceOpenIsProviderAgnostic(t *testing.T) {
	source := Source{
		Kind:   "apple_health_export",
		SHA256: "digest",
		Open: func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(nilReader{}), nil
		},
	}
	reader, err := source.Open(context.Background())
	if err != nil {
		t.Fatalf("Source.Open() error = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() error = %v", err)
	}
}

type nilReader struct{}

func (nilReader) Read([]byte) (int, error) { return 0, io.EOF }
