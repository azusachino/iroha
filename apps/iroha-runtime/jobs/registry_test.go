package jobs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
)

func TestRegisterDecodesTypedPayload(t *testing.T) {
	type payload struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}
	registry := NewRegistry()
	var got payload
	Register(registry, "demo", func(_ context.Context, p payload) error {
		got = p
		return nil
	})
	handler, ok := registry.Handlers()["demo"]
	if !ok {
		t.Fatal("handler not registered")
	}
	job := models.Job{Kind: "demo", PayloadJSON: json.RawMessage(`{"name":"frieren","count":12}`)}
	if err := handler(context.Background(), job); err != nil {
		t.Fatalf("handler: %v", err)
	}
	if got.Name != "frieren" || got.Count != 12 {
		t.Fatalf("decoded = %+v, want frieren/12", got)
	}
}

func TestRegisterEmptyPayloadIsZeroValue(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}
	registry := NewRegistry()
	Register(registry, "empty", func(_ context.Context, p payload) error {
		if p.Name != "" {
			t.Fatalf("want zero payload, got %+v", p)
		}
		return nil
	})
	// nil, {} and null payloads should all decode to the zero value without error.
	for _, raw := range []json.RawMessage{nil, json.RawMessage(`{}`), json.RawMessage(`null`)} {
		if err := registry.Handlers()["empty"](context.Background(), models.Job{PayloadJSON: raw}); err != nil {
			t.Fatalf("handler(%s): %v", raw, err)
		}
	}
}

func TestRegisterInvalidPayloadErrors(t *testing.T) {
	type payload struct {
		Count int `json:"count"`
	}
	registry := NewRegistry()
	Register(registry, "bad", func(_ context.Context, _ payload) error { return nil })
	err := registry.Handlers()["bad"](context.Background(), models.Job{Kind: "bad", PayloadJSON: json.RawMessage(`{"count":"nope"}`)})
	if err == nil {
		t.Fatal("expected decode error for mistyped payload")
	}
}
