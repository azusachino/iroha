package connectors

import (
	"context"
	"testing"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
)

type fakeConnector struct{ id string }

func (f fakeConnector) Descriptor() connector.Descriptor {
	return connector.Descriptor{ID: f.id, SourceKind: f.id}
}

func (fakeConnector) Fetch(context.Context, connector.Credentials, *connector.Cursor) (connector.Snapshot, *connector.Cursor, error) {
	return connector.Snapshot{}, nil, nil
}

func TestRegistrySortsAndFindsConnectors(t *testing.T) {
	registry, err := New(fakeConnector{id: "bangumi"}, fakeConnector{id: "anilist"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got, ok := registry.Get("anilist"); !ok || got.Descriptor().ID != "anilist" {
		t.Fatalf("Get(anilist) = %#v, %v", got, ok)
	}
	descriptors := registry.List()
	if len(descriptors) != 2 || descriptors[0].ID != "anilist" || descriptors[1].ID != "bangumi" {
		t.Fatalf("List() = %#v", descriptors)
	}
}

func TestRegistryRejectsDuplicateIDs(t *testing.T) {
	if _, err := New(fakeConnector{id: "anilist"}, fakeConnector{id: "anilist"}); err == nil {
		t.Fatal("New() accepted duplicate connector IDs")
	}
}
