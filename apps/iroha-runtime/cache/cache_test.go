package cache

import (
	"context"
	"errors"
	"sync"
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

type generationFakeStore struct {
	fakeStore
	generation int64
}

func (s *generationFakeStore) GetWithGeneration(_ context.Context, namespace, key string) ([]byte, int64, bool, error) {
	if s.generation == 0 {
		s.generation = 1
	}
	value, found := s.values[namespace+":"+key]
	return value, s.generation, found, nil
}

func (s *generationFakeStore) SetAtGeneration(_ context.Context, namespace, key string, generation int64, value []byte, _ time.Duration) (bool, error) {
	if s.generation == 0 {
		s.generation = 1
	}
	if generation != s.generation {
		return false, nil
	}
	if s.values == nil {
		s.values = make(map[string][]byte)
	}
	s.values[namespace+":"+key] = value
	return true, nil
}

func (s *generationFakeStore) InvalidateNamespace(_ context.Context, namespace string) error {
	s.generation++
	s.invalidated = namespace
	return nil
}

func TestGenerationAwarePopulationSkipsStaleWrite(t *testing.T) {
	store := &generationFakeStore{}
	c := NewWithStore(store)

	_, generation, found := GetWithGeneration[string](context.Background(), c, "read_reports", "month=2026-08")
	if found || generation != 1 {
		t.Fatalf("initial lookup = found %v, generation %d; want false, 1", found, generation)
	}
	if err := c.InvalidateNamespace(context.Background(), "read_reports"); err != nil {
		t.Fatalf("invalidate namespace: %v", err)
	}
	if stored := SetAtGeneration(context.Background(), c, "read_reports", "month=2026-08", generation, time.Minute, "stale"); stored {
		t.Fatal("stale generation write was stored")
	}
	if _, found := Get[string](context.Background(), c, "read_reports", "month=2026-08"); found {
		t.Fatal("stale value remained in cache")
	}

	_, generation, found = GetWithGeneration[string](context.Background(), c, "read_reports", "month=2026-08")
	if found || generation != 2 {
		t.Fatalf("post-invalidation lookup = found %v, generation %d; want false, 2", found, generation)
	}
	if stored := SetAtGeneration(context.Background(), c, "read_reports", "month=2026-08", generation, time.Minute, "fresh"); !stored {
		t.Fatal("current generation write was not stored")
	}
	if value, found := Get[string](context.Background(), c, "read_reports", "month=2026-08"); !found || value != "fresh" {
		t.Fatalf("current value = %q/%v, want fresh/true", value, found)
	}
}

type invalidationFakeStore struct {
	fakeStore
	failures   int
	attempts   int
	namespaces []string
}

func (s *invalidationFakeStore) InvalidateNamespace(_ context.Context, namespace string) error {
	s.attempts++
	s.namespaces = append(s.namespaces, namespace)
	if s.failures > 0 {
		s.failures--
		return errors.New("cache backend unavailable")
	}
	return nil
}

func TestInvalidateChangeUsesDependencyMatrix(t *testing.T) {
	store := &invalidationFakeStore{}
	c := NewWithStore(store)
	if err := c.InvalidateChange(context.Background(), ChangeExpense); err != nil {
		t.Fatalf("invalidate expense change: %v", err)
	}
	if got, want := store.namespaces, []string{NamespaceExpenses, NamespaceMetrics, NamespaceReports}; !equalStrings(got, want) {
		t.Fatalf("invalidated namespaces = %#v, want %#v", got, want)
	}
	if err := c.InvalidateChange(context.Background(), ChangeKind("unknown")); err == nil {
		t.Fatal("unknown change kind did not fail")
	}
}

func TestInvalidateNamespaceRetriesAndMarksDegraded(t *testing.T) {
	store := &invalidationFakeStore{failures: invalidationAttempts}
	c := NewWithStore(store)
	if err := c.InvalidateNamespace(context.Background(), NamespaceReports); err == nil {
		t.Fatal("persistent invalidation failure did not return an error")
	}
	if store.attempts != invalidationAttempts {
		t.Fatalf("invalidation attempts = %d, want %d", store.attempts, invalidationAttempts)
	}
	if !c.IsDegraded(NamespaceReports) {
		t.Fatal("failed namespace was not marked degraded")
	}
	if c.InvalidationFailureCount() != 1 {
		t.Fatalf("invalidation failure count = %d, want 1", c.InvalidationFailureCount())
	}

	if err := c.InvalidateNamespace(context.Background(), NamespaceReports); err != nil {
		t.Fatalf("recovered invalidation: %v", err)
	}
	if c.IsDegraded(NamespaceReports) {
		t.Fatal("recovered namespace remained degraded")
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func TestValkeyKeysUseApplicationPrefix(t *testing.T) {
	if got, want := generationKey("read_daily"), "iroha:cache:v2:read_daily:__generation"; got != want {
		t.Fatalf("generation key = %q, want %q", got, want)
	}
	if got, want := namespacedKey("read_daily", 3, "GET /api/v1/daily"), "iroha:cache:v2:read_daily:g3:GET /api/v1/daily"; got != want {
		t.Fatalf("entry key = %q, want %q", got, want)
	}
}

func TestGetOrLoad_CoalescesConcurrentMisses(t *testing.T) {
	c := NewWithStore(&fakeStore{})
	var mu sync.Mutex
	calls := 0
	loader := func() (string, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return "loaded", nil
	}

	var wg sync.WaitGroup
	values := make([]string, 2)
	for i := range values {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			value, err := GetOrLoad(context.Background(), c, "public", "same", time.Minute, loader)
			if err != nil {
				t.Errorf("load: %v", err)
				return
			}
			values[index] = value
		}(i)
	}
	wg.Wait()
	if calls != 1 || values[0] != "loaded" || values[1] != "loaded" {
		t.Fatalf("calls = %d, values = %#v", calls, values)
	}
}
