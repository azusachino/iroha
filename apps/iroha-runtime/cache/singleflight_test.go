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

type flightWaitingContext struct {
	context.Context
	once    sync.Once
	waiting chan struct{}
}

func (c *flightWaitingContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func TestGetOrLoadPanicReleasesJoinedWaitersAndAllowsRetry(t *testing.T) {
	client := NewWithStore(&fakeStore{})
	started := make(chan struct{})
	waiterContext := &flightWaitingContext{Context: context.Background(), waiting: make(chan struct{})}
	waiterDone := make(chan struct{})
	var waiterErr error
	var waiterVal string

	leaderDone := make(chan struct{})
	go func() {
		defer close(leaderDone)
		defer func() {
			if recovered := recover(); recovered != "loader panic" {
				t.Errorf("recovered = %v, want loader panic", recovered)
			}
		}()
		_, _ = GetOrLoad(context.Background(), client, "test", "key", time.Minute, func() (string, error) {
			close(started)
			select {
			case <-waiterContext.waiting:
			case <-time.After(time.Second):
				t.Error("waiter did not join flight before timeout")
			}
			panic("loader panic")
		})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("leader did not start")
	}

	go func() {
		defer close(waiterDone)
		waiterVal, waiterErr = GetOrLoad(waiterContext, client, "test", "key", time.Minute, func() (string, error) {
			return "waiter should not run loader", nil
		})
	}()

	select {
	case <-leaderDone:
	case <-time.After(time.Second):
		t.Fatal("leader did not finish panic cleanup")
	}

	select {
	case <-waiterDone:
	case <-time.After(time.Second):
		t.Fatal("joined waiter did not unblock after loader panic")
	}

	if waiterErr == nil || waiterErr.Error() != "cache loader did not complete" {
		t.Fatalf("waiter error = %v, want 'cache loader did not complete' (val=%q)", waiterErr, waiterVal)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	value, err := GetOrLoad(ctx, client, "test", "key", time.Minute, func() (string, error) {
		return "recovered", nil
	})
	if err != nil || value != "recovered" {
		t.Fatalf("retry = %q/%v, want recovered/nil", value, err)
	}
}

func TestGetOrLoadAtGenerationBypassesInitialLookup(t *testing.T) {
	store := &generationFakeStore{}
	client := NewWithStore(store)
	calls := 0
	val, err := GetOrLoadAtGeneration(context.Background(), client, "test", "key", 1, time.Minute, func() (string, error) {
		calls++
		return "val", nil
	})
	if err != nil || val != "val" || calls != 1 {
		t.Fatalf("val=%q err=%v calls=%d", val, err, calls)
	}
}
