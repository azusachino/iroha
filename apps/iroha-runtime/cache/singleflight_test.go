package cache

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type delayedMissStore struct {
	fakeStore
	reads   atomic.Int32
	missed  chan struct{}
	release chan struct{}
}

func (s *delayedMissStore) Get(ctx context.Context, namespace, key string) ([]byte, bool, error) {
	value, found, err := s.fakeStore.Get(ctx, namespace, key)
	if s.reads.Add(1) == 1 {
		close(s.missed)
		<-s.release
	}
	return value, found, err
}

func TestGetOrLoadRechecksCacheAfterDelayedMiss(t *testing.T) {
	store := &delayedMissStore{missed: make(chan struct{}), release: make(chan struct{})}
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(store.release) }) })
	client := NewWithStore(store)
	var calls atomic.Int32
	loader := func() (string, error) {
		calls.Add(1)
		return "loaded", nil
	}
	firstDone := make(chan struct{})
	go func() {
		defer close(firstDone)
		value, err := GetOrLoad(context.Background(), client, "test", "key", time.Minute, loader)
		if err != nil || value != "loaded" {
			t.Errorf("delayed read = %q/%v", value, err)
		}
	}()
	select {
	case <-store.missed:
	case <-time.After(time.Second):
		t.Fatal("first read did not miss")
	}
	value, err := GetOrLoad(context.Background(), client, "test", "key", time.Minute, loader)
	if err != nil || value != "loaded" {
		t.Fatalf("second read = %q/%v", value, err)
	}
	releaseOnce.Do(func() { close(store.release) })
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("delayed read did not return")
	}
	if calls.Load() != 1 {
		t.Fatalf("loader calls = %d, want 1", calls.Load())
	}
}

func TestGetOrLoadPanicReleasesFlightAndAllowsRetry(t *testing.T) {
	client := NewWithStore(&fakeStore{})
	func() {
		defer func() {
			if recovered := recover(); recovered != "loader panic" {
				t.Fatalf("recovered = %v, want loader panic", recovered)
			}
		}()
		_, _ = GetOrLoad(context.Background(), client, "test", "key", time.Minute, func() (string, error) {
			panic("loader panic")
		})
	}()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	value, err := GetOrLoad(ctx, client, "test", "key", time.Minute, func() (string, error) {
		return "recovered", nil
	})
	if err != nil || value != "recovered" {
		t.Fatalf("retry = %q/%v, want recovered/nil", value, err)
	}
}
