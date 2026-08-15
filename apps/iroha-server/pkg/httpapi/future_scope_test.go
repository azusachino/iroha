package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
)

func TestRejectFutureReadScopeRunsBeforeCache(t *testing.T) {
	store := &readCacheTestStore{}
	server := &Server{
		deps: Dependencies{
			Config: config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}},
			Cache:  cache.NewWithStore(store),
		},
		now: func() time.Time {
			return time.Date(2026, time.August, 15, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
		},
	}
	calls := 0
	handler := server.rejectFutureReadScope(server.readCache(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		writeJSON(w, http.StatusOK, map[string]string{"value": "loaded"})
	})))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/briefing?date=2099-01-01", nil))

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
	if calls != 0 {
		t.Fatalf("handler calls = %d, want 0", calls)
	}
	if got := response.Header().Get("X-Iroha-Cache"); got != "" {
		t.Fatalf("cache header = %q, want empty", got)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("cache-control = %q, want no-store", got)
	}
}

func TestFutureReadScopeAllowsCurrentPeriodEnvelope(t *testing.T) {
	server := &Server{
		deps: Dependencies{Config: config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}}},
		now: func() time.Time {
			return time.Date(2026, time.August, 15, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
		},
	}
	calls := 0
	handler := server.rejectFutureReadScope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/metrics/health.steps/series?from=2026-08-01&to=2026-09-01&grain=month", nil))

	if response.Code != http.StatusNoContent || calls != 1 {
		t.Fatalf("response = %d, calls = %d, want 204/1", response.Code, calls)
	}
}

// TestFutureReadScopeAllowsParamFreeBoundsRequests confirms every new
// /bounds route -- which takes no date/from/to/scope param and so can never
// itself resolve to a future scope -- passes straight through the guard
// unaffected, the same way a health check would.
func TestFutureReadScopeAllowsParamFreeBoundsRequests(t *testing.T) {
	server := &Server{
		deps: Dependencies{Config: config.Config{Server: config.ServerConfig{Timezone: "Asia/Tokyo"}}},
		now: func() time.Time {
			return time.Date(2026, time.August, 15, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
		},
	}
	for _, path := range []string{
		"/api/v1/activities/bounds",
		"/api/v1/sleep/bounds",
		"/api/v1/expenses/bounds",
		"/api/v1/daily/bounds",
		"/api/v1/activities/bounds?timezone=Asia/Tokyo",
	} {
		t.Run(path, func(t *testing.T) {
			calls := 0
			handler := server.rejectFutureReadScope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(http.StatusNoContent)
			}))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusNoContent || calls != 1 {
				t.Fatalf("response = %d, calls = %d, want 204/1", response.Code, calls)
			}
		})
	}
}
