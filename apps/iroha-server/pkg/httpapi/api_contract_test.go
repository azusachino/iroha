package httpapi

import (
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestActiveRouteInventory keeps the active OpenAPI surface aligned with the
// routes registered by the server. Deferred roadmap resources must not enter
// the active contract until their handlers exist.
func TestActiveRouteInventory(t *testing.T) {
	server := &Server{mux: chi.NewRouter()}
	server.routes()

	got := make([]string, 0)
	if err := chi.Walk(server.mux, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		got = append(got, method+" "+normalizeRoute(route))
		return nil
	}); err != nil {
		t.Fatalf("walk routes: %v", err)
	}
	sort.Strings(got)

	want := []string{
		"GET /api/v1/activities",
		"GET /api/v1/activities/routes",
		"GET /api/v1/activities/summary",
		"GET /api/v1/activities/{activityId}",
		"GET /api/v1/activities/{activityId}/laps",
		"GET /api/v1/activities/{activityId}/route",
		"GET /api/v1/activities/{activityId}/samplings",
		"GET /api/v1/briefing",
		"GET /api/v1/daily",
		"GET /api/v1/daily/aggregates",
		"GET /api/v1/imports",
		"GET /api/v1/imports/{importId}",
		"GET /api/v1/media",
		"GET /api/v1/media/aggregates",
		"GET /api/v1/media/events",
		"GET /api/v1/media/{mediaId}",
		"GET /api/v1/raw-files",
		"GET /api/v1/raw-files/{rawFileId}",
		"GET /api/v1/sleep",
		"GET /api/v1/sleep/aggregates",
		"GET /api/v1/sleep/{sleepId}/segments",
		"GET /healthz",
		"POST /api/v1/imports",
		"POST /api/v1/media/sync/{connectorId}",
		"POST /api/v1/raw-files",
	}
	sort.Strings(want)

	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("active route inventory differs\nwant:\n%s\ngot:\n%s", strings.Join(want, "\n"), strings.Join(got, "\n"))
	}
}

func normalizeRoute(route string) string {
	if route != "/" {
		route = strings.TrimSuffix(route, "/")
	}
	return route
}
