package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestGetOrLoad_DisabledClient_CallsLoaderOnce(t *testing.T) {
	c := New("")

	calls := 0
	loader := func() (string, error) {
		calls++
		return "loaded-value", nil
	}

	ctx := context.Background()
	value, err := GetOrLoad(ctx, c, "test", "some-key", time.Minute, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "loaded-value" {
		t.Fatalf("got %q, want %q", value, "loaded-value")
	}
	if calls != 1 {
		t.Fatalf("loader called %d times, want 1", calls)
	}
}

func TestGetOrLoad_UnreachableRedis_FallsBackToLoader(t *testing.T) {
	// Port with nothing listening: redis calls should fail fast rather than
	// hang, thanks to the short context timeout below.
	c := New("redis://127.0.0.1:6390/0")

	calls := 0
	loader := func() (int, error) {
		calls++
		return 42, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	value, err := GetOrLoad(ctx, c, "test", "another-key", time.Minute, loader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != 42 {
		t.Fatalf("got %d, want 42", value)
	}
	if calls != 1 {
		t.Fatalf("loader called %d times, want 1", calls)
	}
}

func TestGetOrLoad_LoaderError_Propagates(t *testing.T) {
	c := New("")

	wantErr := errors.New("boom")
	loader := func() (string, error) {
		return "", wantErr
	}

	ctx := context.Background()
	_, err := GetOrLoad(ctx, c, "test", "err-key", time.Minute, loader)
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
}

type fakeStore struct {
	values      map[string][]byte
	namespace   string
	invalidated string
}

func (s *fakeStore) Get(_ context.Context, namespace, key string) ([]byte, bool, error) {
	value, ok := s.values[namespace+":"+key]
	return value, ok, nil
}

func (s *fakeStore) Set(_ context.Context, namespace, key string, value []byte, _ time.Duration) error {
	if s.values == nil {
		s.values = make(map[string][]byte)
	}
	s.namespace = namespace
	s.values[namespace+":"+key] = value
	return nil
}

func (s *fakeStore) InvalidateNamespace(_ context.Context, namespace string) error {
	s.invalidated = namespace
	return nil
}

func (s *fakeStore) Close() error { return nil }

func TestClient_UsesBackendNamespaceContract(t *testing.T) {
	store := &fakeStore{}
	c := NewWithStore(store)

	Set(context.Background(), c, "public_summary", "v1:2024", time.Minute, "cached")
	value, ok := Get[string](context.Background(), c, "public_summary", "v1:2024")
	if !ok || value != "cached" {
		t.Fatalf("got %q/%v, want cached/true", value, ok)
	}
	if store.namespace != "public_summary" {
		t.Fatalf("namespace = %q, want public_summary", store.namespace)
	}
	if err := c.InvalidateNamespace(context.Background(), "public_summary"); err != nil {
		t.Fatalf("invalidate namespace: %v", err)
	}
	if store.invalidated != "public_summary" {
		t.Fatalf("invalidated namespace = %q, want public_summary", store.invalidated)
	}
}
