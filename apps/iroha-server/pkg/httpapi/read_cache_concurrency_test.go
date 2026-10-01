package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
)

// Done is observed only when the cache joins a flight in these DB-free tests.
type cacheWaitingContext struct {
	context.Context
	once    sync.Once
	waiting chan struct{}
}

func (c *cacheWaitingContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.waiting) })
	return c.Context.Done()
}

func awaitCacheSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Fatal("cache request did not reach the expected barrier")
	}
}

func TestReadCacheCoalescesConcurrentMisses(t *testing.T) {
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&generationReadCacheTestStore{})}}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var calls atomic.Int32
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		writeJSON(w, http.StatusOK, map[string]string{"value": "loaded"})
	}))
	first, second := httptest.NewRecorder(), httptest.NewRecorder()
	first.Header().Set("X-Request-ID", "owner")
	second.Header().Set("X-Request-ID", "waiter")
	firstDone, secondDone := make(chan struct{}), make(chan struct{})
	go func() {
		handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		close(firstDone)
	}()
	awaitCacheSignal(t, started)
	ctx := &cacheWaitingContext{Context: context.Background(), waiting: make(chan struct{})}
	go func() {
		handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil).WithContext(ctx))
		close(secondDone)
	}()
	awaitCacheSignal(t, ctx.waiting)
	releaseOnce.Do(func() { close(release) })
	awaitCacheSignal(t, firstDone)
	awaitCacheSignal(t, secondDone)
	if calls.Load() != 1 || first.Body.String() != second.Body.String() || second.Code != http.StatusOK {
		t.Fatalf("calls=%d, responses=%s/%s, status=%d", calls.Load(), first.Body.String(), second.Body.String(), second.Code)
	}
	if second.Header().Get("X-Request-ID") != "waiter" {
		t.Fatalf("waiter request ID = %q", second.Header().Get("X-Request-ID"))
	}
}

func TestReadCacheCanceledWaiterDoesNotRunHandler(t *testing.T) {
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&generationReadCacheTestStore{})}}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var calls atomic.Int32
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		writeJSON(w, http.StatusOK, map[string]string{"value": "loaded"})
	}))
	firstDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		close(firstDone)
	}()
	awaitCacheSignal(t, started)
	canceled, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &cacheWaitingContext{Context: canceled, waiting: make(chan struct{})}
	secondDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil).WithContext(ctx))
		close(secondDone)
	}()
	awaitCacheSignal(t, ctx.waiting)
	cancel()
	awaitCacheSignal(t, secondDone)
	if calls.Load() != 1 {
		t.Fatalf("handler calls=%d, want 1", calls.Load())
	}
	releaseOnce.Do(func() { close(release) })
	awaitCacheSignal(t, firstDone)
}

func TestReadCacheInvalidationSeparatesInFlightLoads(t *testing.T) {
	client := cache.NewWithStore(&generationReadCacheTestStore{})
	server := &Server{deps: Dependencies{Cache: client}}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var calls atomic.Int32
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			close(started)
			<-release
		}
		writeJSON(w, http.StatusOK, map[string]int32{"call": call})
	}))
	firstDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		close(firstDone)
	}()
	awaitCacheSignal(t, started)
	if err := client.InvalidateNamespace(context.Background(), cache.NamespaceDaily); err != nil {
		t.Fatal(err)
	}
	fresh := httptest.NewRecorder()
	freshDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(fresh, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		close(freshDone)
	}()
	awaitCacheSignal(t, freshDone)
	releaseOnce.Do(func() { close(release) })
	awaitCacheSignal(t, firstDone)
	cached := httptest.NewRecorder()
	handler.ServeHTTP(cached, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
	if calls.Load() != 2 || fresh.Body.String() != cached.Body.String() || cached.Header().Get("X-Iroha-Cache") != "HIT" {
		t.Fatalf("calls=%d, fresh=%s cached=%s header=%s", calls.Load(), fresh.Body.String(), cached.Body.String(), cached.Header().Get("X-Iroha-Cache"))
	}
}

func TestReadCacheReloadsEvictedOrMalformedEntries(t *testing.T) {
	for _, malformed := range []bool{false, true} {
		store := &generationReadCacheTestStore{}
		server := &Server{deps: Dependencies{Cache: cache.NewWithStore(store)}}
		calls := 0
		handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls++
			writeJSON(w, http.StatusOK, map[string]int{"call": calls})
		}))
		handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		// Expiry/eviction removes an entry without advancing the generation.
		store.mu.Lock()
		for key := range store.values {
			if malformed {
				store.values[key] = []byte("malformed cache envelope")
			} else {
				delete(store.values, key)
			}
		}
		store.mu.Unlock()
		refreshed, cached := httptest.NewRecorder(), httptest.NewRecorder()
		handler.ServeHTTP(refreshed, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		handler.ServeHTTP(cached, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		if calls != 2 || refreshed.Body.String() != cached.Body.String() || refreshed.Header().Get("X-Iroha-Cache") != "MISS" || cached.Header().Get("X-Iroha-Cache") != "HIT" {
			t.Fatalf("malformed=%t calls=%d refreshed=%s cached=%s", malformed, calls, refreshed.Body.String(), cached.Body.String())
		}
	}
}

func TestReadCacheDoesNotCacheUnsuccessfulOrNonJSONResponses(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		status      int
		contentType string
		body        string
	}{
		{"non-JSON", http.StatusOK, "text/plain", "not cacheable"},
		{"client error", http.StatusBadRequest, "application/json", `{"error":"bad request"}`},
		{"server error", http.StatusInternalServerError, "application/json", `{"error":"unavailable"}`},
		{"empty JSON", http.StatusOK, "application/json", ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&generationReadCacheTestStore{})}}
			calls := 0
			handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.Header().Set("Content-Type", testCase.contentType)
				w.WriteHeader(testCase.status)
				_, _ = w.Write([]byte(testCase.body))
			}))
			for range 2 {
				response := httptest.NewRecorder()
				handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
				if response.Code != testCase.status || response.Body.String() != testCase.body {
					t.Fatalf("response=%d/%s", response.Code, response.Body.String())
				}
			}
			if calls != 2 {
				t.Fatalf("handler calls=%d, want 2", calls)
			}
		})
	}
}

type countingReadCacheTestStore struct {
	generationReadCacheTestStore
	lookups atomic.Int32
}

func (s *countingReadCacheTestStore) GetWithGeneration(ctx context.Context, namespace, key string) ([]byte, int64, bool, error) {
	s.lookups.Add(1)
	return s.generationReadCacheTestStore.GetWithGeneration(ctx, namespace, key)
}

func TestReadCacheAvoidsRedundantLookupsOnConcurrentMiss(t *testing.T) {
	store := &countingReadCacheTestStore{}
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(store)}}
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })

	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		writeJSON(w, http.StatusOK, map[string]string{"result": "ok"})
	}))

	first, second := httptest.NewRecorder(), httptest.NewRecorder()
	firstDone, secondDone := make(chan struct{}), make(chan struct{})

	go func() {
		handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		close(firstDone)
	}()
	awaitCacheSignal(t, started)

	ctx := &cacheWaitingContext{Context: context.Background(), waiting: make(chan struct{})}
	go func() {
		handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil).WithContext(ctx))
		close(secondDone)
	}()
	awaitCacheSignal(t, ctx.waiting)

	releaseOnce.Do(func() { close(release) })
	awaitCacheSignal(t, firstDone)
	awaitCacheSignal(t, secondDone)

	// In the redundant-lookup implementation (calling GetOrLoad without generation reuse),
	// request 1 does: server.go (1) + GetOrLoad (1) + post-flight recheck (1) = 3 lookups.
	// request 2 does: server.go (1) + GetOrLoad (1) = 2 lookups. Total was 5 lookups.
	// In the fixed implementation (GetOrLoadAtGeneration), request 1 does: server.go (1) + recheck (1) = 2.
	// request 2 does: server.go (1) + 0 in GetOrLoadAtGeneration = 1. Total is 3 lookups.
	if lookups := store.lookups.Load(); lookups != 3 {
		t.Fatalf("cache lookups = %d, want exactly 3 (redundant initial lookups would cause 5)", lookups)
	}
}

func TestReadCacheIndependentKeysAndRevisionsProceedConcurrently(t *testing.T) {
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&generationReadCacheTestStore{})}}
	dailyStarted, expensesStarted := make(chan struct{}), make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })

	var dailyCalls, expensesCalls atomic.Int32
	dailyHandler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		dailyCalls.Add(1)
		close(dailyStarted)
		<-release
		writeJSON(w, http.StatusOK, map[string]string{"type": "daily"})
	}))
	expensesHandler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		expensesCalls.Add(1)
		close(expensesStarted)
		<-release
		writeJSON(w, http.StatusOK, map[string]string{"type": "expenses"})
	}))

	dailyDone, expensesDone := make(chan struct{}), make(chan struct{})
	go func() {
		dailyHandler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		close(dailyDone)
	}()
	go func() {
		expensesHandler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/expenses", nil))
		close(expensesDone)
	}()

	awaitCacheSignal(t, dailyStarted)
	awaitCacheSignal(t, expensesStarted)

	releaseOnce.Do(func() { close(release) })
	awaitCacheSignal(t, dailyDone)
	awaitCacheSignal(t, expensesDone)

	if dailyCalls.Load() != 1 || expensesCalls.Load() != 1 {
		t.Fatalf("calls = daily:%d expenses:%d, want 1 each", dailyCalls.Load(), expensesCalls.Load())
	}
}

func TestReadCacheConcurrentFailedResponsesRetainOwnContext(t *testing.T) {
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&generationReadCacheTestStore{})}}
	leaderStarted, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })

	var calls atomic.Int32
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			w.Header().Set("X-Failed-Role", "leader")
			close(leaderStarted)
			<-release
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"leader failed"}`))
			return
		}
		w.Header().Set("X-Failed-Role", "waiter")
		writeJSON(w, http.StatusOK, map[string]string{"role": "waiter-recovered"})
	}))

	leaderRec, waiterRec := httptest.NewRecorder(), httptest.NewRecorder()
	leaderDone, waiterDone := make(chan struct{}), make(chan struct{})

	go func() {
		handler.ServeHTTP(leaderRec, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
		close(leaderDone)
	}()
	awaitCacheSignal(t, leaderStarted)

	ctx := &cacheWaitingContext{Context: context.Background(), waiting: make(chan struct{})}
	go func() {
		handler.ServeHTTP(waiterRec, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil).WithContext(ctx))
		close(waiterDone)
	}()
	awaitCacheSignal(t, ctx.waiting)

	releaseOnce.Do(func() { close(release) })
	awaitCacheSignal(t, leaderDone)
	awaitCacheSignal(t, waiterDone)

	if leaderRec.Code != http.StatusBadRequest || leaderRec.Header().Get("X-Failed-Role") != "leader" {
		t.Fatalf("leader = code %d role %s", leaderRec.Code, leaderRec.Header().Get("X-Failed-Role"))
	}
	if waiterRec.Code != http.StatusOK || waiterRec.Header().Get("X-Failed-Role") != "waiter" {
		t.Fatalf("waiter did not retain own context: code %d role %s body %s", waiterRec.Code, waiterRec.Header().Get("X-Failed-Role"), waiterRec.Body.String())
	}
	cachedRec := httptest.NewRecorder()
	handler.ServeHTTP(cachedRec, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
	if calls.Load() != 3 {
		t.Fatalf("handler calls = %d, want 3 (failed response must not be cached)", calls.Load())
	}
}

func TestReadCacheLeaderPanicReleasesJoinedWaitersAndFailsOpen(t *testing.T) {
	server := &Server{deps: Dependencies{Cache: cache.NewWithStore(&generationReadCacheTestStore{})}}
	leaderStarted, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })

	var calls atomic.Int32
	handler := server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		call := calls.Add(1)
		if call == 1 {
			close(leaderStarted)
			<-release
			panic("simulated handler panic")
		}
		writeJSON(w, http.StatusOK, map[string]string{"recovered": "ok"})
	}))

	leaderDone, waiterDone := make(chan struct{}), make(chan struct{})
	leaderRec, waiterRec := httptest.NewRecorder(), httptest.NewRecorder()

	ctx := &cacheWaitingContext{Context: context.Background(), waiting: make(chan struct{})}

	go func() {
		defer close(leaderDone)
		defer func() {
			_ = recover()
		}()
		handler.ServeHTTP(leaderRec, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
	}()

	awaitCacheSignal(t, leaderStarted)

	go func() {
		handler.ServeHTTP(waiterRec, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil).WithContext(ctx))
		close(waiterDone)
	}()
	awaitCacheSignal(t, ctx.waiting)

	releaseOnce.Do(func() { close(release) })
	awaitCacheSignal(t, leaderDone)
	awaitCacheSignal(t, waiterDone)

	if waiterRec.Code != http.StatusOK {
		t.Fatalf("waiter code = %d, want 200 (fail-open)", waiterRec.Code)
	}

	retryRec := httptest.NewRecorder()
	handler.ServeHTTP(retryRec, httptest.NewRequest(http.MethodGet, "/api/v1/daily", nil))
	if retryRec.Code != http.StatusOK {
		t.Fatalf("retry code = %d, want 200", retryRec.Code)
	}
}
