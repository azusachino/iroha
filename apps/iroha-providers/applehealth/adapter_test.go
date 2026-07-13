package applehealth

import (
	"context"
	"errors"
	"strings"
	"testing"

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
	_, err := New().ImportActivities(context.Background(), provider.Source{Kind: "zip"}, provider.ImportOptions{Requested: []provider.Capability{provider.CapabilityActivities}})
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
	_, err := (contextReader{ctx: ctx, reader: strings.NewReader("data")}).Read(make([]byte, 4))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Read() error = %v", err)
	}
}
