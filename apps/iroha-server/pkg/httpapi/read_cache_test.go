package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
)

type readCacheTestStore struct {
	mu         sync.Mutex
	values     map[string][]byte
	generation map[string]int
}

func (s *readCacheTestStore) Get(_ context.Context, namespace, key string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.values[s.cacheKey(namespace, key)]
	return value, ok, nil
}

func (s *readCacheTestStore) Set(_ context.Context, namespace, key string, value []byte, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.values == nil {
		s.values = make(map[string][]byte)
	}
	s.values[s.cacheKey(namespace, key)] = append([]byte(nil), value...)
	return nil
}

func (s *readCacheTestStore) InvalidateNamespace(_ context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.generation == nil {
		s.generation = make(map[string]int)
	}
	s.generation[namespace]++
	for key := range s.values {
		if len(key) >= len(namespace)+1 && key[:len(namespace)+1] == namespace+":" {
			delete(s.values, key)
		}
	}
	return nil
}

func (s *readCacheTestStore) Close() error { return nil }

func (s *readCacheTestStore) cacheKey(namespace, key string) string {
	return namespace + ":" + key
}

func TestReadCacheCachesSuccessfulJSONReads(t *testing.T) {
	store := &readCacheTestStore{}
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(store)}}
	calls := 0
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, http.StatusOK, map[string]string{"value": "loaded"})
	}))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/daily?from=2026-01-01&to=2026-01-31", nil))
	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/daily?to=2026-01-31&from=2026-01-01", nil))

	if calls != 1 {
		t.Fatalf("handler calls = %d, want 1", calls)
	}
	if first.Header().Get("X-Iroha-Cache") != "MISS" || second.Header().Get("X-Iroha-Cache") != "HIT" {
		t.Fatalf("cache headers = %q/%q, want MISS/HIT", first.Header().Get("X-Iroha-Cache"), second.Header().Get("X-Iroha-Cache"))
	}
	if second.Body.String() != first.Body.String() {
		t.Fatalf("cached body = %q, want %q", second.Body.String(), first.Body.String())
	}
}

func TestReadCacheInvalidationReloadsData(t *testing.T) {
	store := &readCacheTestStore{}
	client := cache.NewWithStore(store)
	server := &Server{deps: Dependencies{Cache: client}}
	calls := 0
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, http.StatusOK, map[string]int{"call": calls})
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/activities", nil)

	handler.ServeHTTP(httptest.NewRecorder(), request)
	if err := client.InvalidateNamespace(context.Background(), cache.NamespaceActivities); err != nil {
		t.Fatalf("invalidate cache: %v", err)
	}
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2 after invalidation", calls)
	}
}

func TestReadCacheSkipsMutationsAndUnrelatedPaths(t *testing.T) {
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&readCacheTestStore{})}}
	for _, method := range []string{http.MethodPost, http.MethodGet} {
		for _, path := range []string{"/api/v1/media/sync/anilist", "/api/v1/activitiesfoo", "/api/v1/expenses", "/api/v1/reports/monthly"} {
			request := httptest.NewRequest(method, path, nil)
			namespace, ok := readCacheNamespace(request)
			if ok {
				t.Fatalf("%s %s classified as cacheable namespace %q", method, path, namespace)
			}
		}
	}
	_ = server
}
