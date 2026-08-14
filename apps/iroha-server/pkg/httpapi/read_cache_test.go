package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
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

type generationReadCacheTestStore struct {
	readCacheTestStore
	current map[string]int64
}

type failingInvalidationReadCacheStore struct {
	readCacheTestStore
}

func (s *failingInvalidationReadCacheStore) InvalidateNamespace(context.Context, string) error {
	return errors.New("cache backend unavailable")
}

func (s *generationReadCacheTestStore) GetWithGeneration(_ context.Context, namespace, key string) ([]byte, int64, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		s.current = make(map[string]int64)
	}
	if s.current[namespace] == 0 {
		s.current[namespace] = 1
	}
	value, found := s.values[s.cacheKey(namespace, key)]
	return value, s.current[namespace], found, nil
}

func (s *generationReadCacheTestStore) SetAtGeneration(_ context.Context, namespace, key string, generation int64, value []byte, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		s.current = make(map[string]int64)
	}
	if s.current[namespace] == 0 {
		s.current[namespace] = 1
	}
	if s.current[namespace] != generation {
		return false, nil
	}
	if s.values == nil {
		s.values = make(map[string][]byte)
	}
	s.values[s.cacheKey(namespace, key)] = append([]byte(nil), value...)
	return true, nil
}

func (s *generationReadCacheTestStore) InvalidateNamespace(_ context.Context, namespace string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil {
		s.current = make(map[string]int64)
	}
	if s.current[namespace] == 0 {
		s.current[namespace] = 1
	}
	s.current[namespace]++
	for key := range s.values {
		if len(key) >= len(namespace)+1 && key[:len(namespace)+1] == namespace+":" {
			delete(s.values, key)
		}
	}
	return nil
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

func TestReadCacheDoesNotPopulateAfterInvalidation(t *testing.T) {
	store := &generationReadCacheTestStore{}
	client := cache.NewWithStore(store)
	server := &Server{deps: Dependencies{Cache: client}}
	started := make(chan struct{})
	release := make(chan struct{})
	calls := 0
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			close(started)
			<-release
		}
		writeJSON(w, http.StatusOK, map[string]int{"call": calls})
	}))

	firstDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/reports/monthly?month=2026-08", nil))
		firstDone <- response
	}()
	<-started
	if err := client.InvalidateNamespace(context.Background(), cache.NamespaceReports); err != nil {
		t.Fatalf("invalidate reports: %v", err)
	}
	close(release)
	<-firstDone

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/reports/monthly?month=2026-08", nil))
	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2 after racing invalidation", calls)
	}
	if second.Header().Get("X-Iroha-Cache") != "MISS" {
		t.Fatalf("second cache header = %q, want MISS", second.Header().Get("X-Iroha-Cache"))
	}
}

func TestReadCacheBypassesDegradedNamespace(t *testing.T) {
	client := cache.NewWithStore(&failingInvalidationReadCacheStore{})
	if err := client.InvalidateNamespace(context.Background(), cache.NamespaceReports); err == nil {
		t.Fatal("persistent invalidation failure did not return an error")
	}
	server := &Server{deps: Dependencies{Cache: client}}
	calls := 0
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, http.StatusOK, map[string]int{"call": calls})
	}))
	for range 2 {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/reports/monthly?month=2026-08", nil))
		if response.Header().Get("X-Iroha-Cache") != "BYPASS" {
			t.Fatalf("cache header = %q, want BYPASS", response.Header().Get("X-Iroha-Cache"))
		}
	}
	if calls != 2 {
		t.Fatalf("handler calls = %d, want 2 while namespace is degraded", calls)
	}
}

func TestReadCacheCachesReportResponses(t *testing.T) {
	store := &readCacheTestStore{}
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(store)}}
	calls := 0
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, http.StatusOK, map[string]int{"call": calls})
	}))

	for _, path := range []string{
		"/api/v1/reports/monthly?month=2026-08",
		"/api/v1/reports/monthly-series?end=2026-08&months=12",
	} {
		first := httptest.NewRecorder()
		handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, path, nil))
		second := httptest.NewRecorder()
		handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, path, nil))

		if first.Header().Get("X-Iroha-Cache") != "MISS" || second.Header().Get("X-Iroha-Cache") != "HIT" {
			t.Fatalf("%s cache headers = %q/%q, want MISS/HIT", path, first.Header().Get("X-Iroha-Cache"), second.Header().Get("X-Iroha-Cache"))
		}
		if first.Body.String() != second.Body.String() {
			t.Fatalf("%s cached body = %q, want %q", path, second.Body.String(), first.Body.String())
		}
	}
	if calls != 2 {
		t.Fatalf("handler calls = %d, want one load per report path", calls)
	}
}

func TestReadCacheSkipsMutationsAndUnrelatedPaths(t *testing.T) {
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&readCacheTestStore{})}}
	for _, method := range []string{http.MethodPost, http.MethodGet} {
		for _, path := range []string{"/api/v1/media/sync/anilist", "/api/v1/activitiesfoo", "/api/v1/expenses"} {
			request := httptest.NewRequest(method, path, nil)
			namespace, ok := readCacheNamespace(request)
			if ok {
				t.Fatalf("%s %s classified as cacheable namespace %q", method, path, namespace)
			}
		}
	}
	for _, path := range []string{"/api/v1/reports/monthly", "/api/v1/reports/monthly-series"} {
		namespace, ok := readCacheNamespace(httptest.NewRequest(http.MethodGet, path, nil))
		if !ok || namespace != cache.NamespaceReports {
			t.Fatalf("GET %s classified as %q/%t, want %q/true", path, namespace, ok, cache.NamespaceReports)
		}
	}
	_ = server
}

func TestReadCacheKeyUsesEffectiveTimezone(t *testing.T) {
	server := &Server{deps: Dependencies{Config: config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}}}}
	omitted := server.readCacheKey(httptest.NewRequest(http.MethodGet, "/api/v1/reports/monthly?month=2026-08", nil))
	explicit := server.readCacheKey(httptest.NewRequest(http.MethodGet, "/api/v1/reports/monthly?timezone=Asia%2FTokyo&month=2026-08", nil))
	utc := server.readCacheKey(httptest.NewRequest(http.MethodGet, "/api/v1/reports/monthly?timezone=UTC&month=2026-08", nil))
	if omitted != explicit {
		t.Fatalf("omitted timezone key = %q, explicit key = %q; want equal", omitted, explicit)
	}
	if omitted == utc {
		t.Fatalf("Tokyo key = %q, UTC key = %q; want different", omitted, utc)
	}
}
