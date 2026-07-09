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
	value, err := GetOrLoad(ctx, c, "some-key", time.Minute, loader)
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

	value, err := GetOrLoad(ctx, c, "another-key", time.Minute, loader)
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
	_, err := GetOrLoad(ctx, c, "err-key", time.Minute, loader)
	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want %v", err, wantErr)
	}
}
