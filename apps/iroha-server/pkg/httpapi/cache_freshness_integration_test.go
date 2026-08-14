//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/mediaresolution"
	"github.com/google/uuid"
)

func TestIntegrationCacheFreshnessAfterExpenseMutations(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	responseCache := cache.NewWithStore(&readCacheTestStore{})
	server := newIntegrationServerWithCache(t, db, responseCache)
	path := "/api/v1/metrics/expenses.amount_minor/series?from=2026-08-01&to=2026-09-01&grain=month&timezone=UTC&dimension=currency%3AJPY"

	first, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("first expense metric cache header = %q, want MISS", header)
	}
	assertMetricMinor(t, first, nil)
	_, header = requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "HIT" {
		t.Fatalf("second expense metric cache header = %q, want HIT", header)
	}

	created := requestJSON(t, server, http.MethodPost, "/api/v1/expenses", `{"occurred_on":"2026-08-12","currency":"JPY","amount_minor":1300,"category":"food","merchant":"Ramen","source":{"kind":"cache-test","ref":"freshness-1"}}`, http.StatusCreated, nil)
	expenseID := stringValue(t, created, "id")
	createdSeries, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("post-create expense metric cache header = %q, want MISS", header)
	}
	assertMetricMinor(t, createdSeries, int64Pointer(1300))

	requestJSON(t, server, http.MethodPut, "/api/v1/expenses/"+expenseID, `{"occurred_on":"2026-08-12","currency":"JPY","amount_minor":1800,"category":"food","merchant":"Ramen"}`, http.StatusOK, nil)
	replacedSeries, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("post-replace expense metric cache header = %q, want MISS", header)
	}
	assertMetricMinor(t, replacedSeries, int64Pointer(1800))

	requestStatus(t, server, http.MethodDelete, "/api/v1/expenses/"+expenseID, http.StatusNoContent)
	deletedSeries, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("post-delete expense metric cache header = %q, want MISS", header)
	}
	assertMetricMinor(t, deletedSeries, nil)
}

func TestIntegrationCacheFreshnessAfterGeocodeRefresh(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	responseCache := cache.NewWithStore(&readCacheTestStore{})
	server := newIntegrationServerWithCache(t, db, responseCache)

	now := time.Date(2099, time.August, 14, 12, 0, 0, 0, time.UTC)
	rawID := uuid.New()
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: "cache-freshness-" + rawID.String(), OriginalFilename: "cache-test.gpx",
		StoragePath: "/tmp/cache-test.gpx", SourceKind: "test", UploadedVia: "test", CreatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	activityID := uuid.New()
	distance := 500.0
	duration := 1800
	if err := db.Create(&models.Activity{
		ID: activityID, SportType: "run", Title: "cache route", StartedAt: now,
		Timezone: "UTC", DistanceM: &distance, DurationS: &duration, SourceKind: "test",
		FirstRawFileID: rawID, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}
	for seq, point := range []struct{ lat, lon float64 }{{35.681236, 139.767125}, {35.682000, 139.768000}} {
		if err := db.Exec(`insert into tb_activity_route_points (activity_id, seq, lat, lon, geom)
			values (?, ?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)`, activityID, seq, point.lat, point.lon, point.lon, point.lat).Error; err != nil {
			t.Fatalf("create route point %d: %v", seq, err)
		}
	}

	path := "/api/v1/activities/routes"
	first, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("first route cache header = %q, want MISS", header)
	}
	assertRouteCity(t, first, "Unknown")
	_, header = requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "HIT" {
		t.Fatalf("second route cache header = %q, want HIT", header)
	}

	key := geocode.CoordinateKey(35.681236, 139.767125)
	if result := db.Exec(`update tb_geocode_cache
		set city = ?, response_json = '{}'::jsonb, fetched_at = ?, expires_at = ?, refresh_queued_at = null, last_error = null, updated_at = ?
		where coordinate_key = ?`, "Tokyo", now, now.Add(365*24*time.Hour), now, key); result.Error != nil {
		t.Fatalf("complete geocode refresh: %v", result.Error)
	} else if result.RowsAffected != 1 {
		t.Fatalf("geocode refresh rows = %d, want 1", result.RowsAffected)
	}
	if err := responseCache.InvalidateChange(context.Background(), cache.ChangeGeocode); err != nil {
		t.Fatalf("invalidate after geocode refresh: %v", err)
	}

	refreshed, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("refreshed route cache header = %q, want MISS", header)
	}
	assertRouteCity(t, refreshed, "Tokyo")
}

func TestIntegrationCacheFreshnessAfterMediaResolution(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	responseCache := cache.NewWithStore(&readCacheTestStore{})
	server := newIntegrationServerWithCache(t, db, responseCache)
	taskID := uuid.New()
	if err := db.Create(&models.MediaResolutionTask{
		ID: taskID, TaskType: "dedupe_candidate", Status: mediaresolution.StatusOpen,
		CandidatesJSON: json.RawMessage(`{"candidates":["candidate-1"]}`),
		ResolutionJSON: json.RawMessage(`{}`), CreatedAt: time.Date(2099, 8, 14, 12, 0, 0, 0, time.UTC),
	}).Error; err != nil {
		t.Fatalf("seed resolution task: %v", err)
	}
	t.Cleanup(func() { _ = db.Delete(&models.MediaResolutionTask{}, "id = ?", taskID).Error })

	path := "/api/v1/media/resolution-tasks"
	first, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("first resolution cache header = %q, want MISS", header)
	}
	assertResolutionTaskStatus(t, first, mediaresolution.StatusOpen)
	_, header = requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "HIT" {
		t.Fatalf("second resolution cache header = %q, want HIT", header)
	}

	taskIDValue := ids.Encode(ids.MediaResolutionTaskPrefix, taskID)
	requestJSON(t, server, http.MethodPatch, "/api/v1/media/resolution-tasks/"+taskIDValue, `{"status":"resolved","resolution":{"decision":"canonical"}}`, http.StatusOK, nil)
	refreshed, header := requestCachedJSON(t, server, http.MethodGet, path, "", http.StatusOK)
	if header != "MISS" {
		t.Fatalf("refreshed resolution cache header = %q, want MISS", header)
	}
	if items := refreshed["items"].([]any); len(items) != 0 {
		t.Fatalf("open resolution items after mutation = %#v, want empty", items)
	}
}

func requestCachedJSON(t *testing.T, handler http.Handler, method, path, body string, wantStatus int) (map[string]any, string) {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != wantStatus {
		t.Fatalf("%s %s status = %d, want %d body = %s", method, path, rec.Code, wantStatus, rec.Body.String())
	}
	var response any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v body = %s", err, rec.Body.String())
	}
	normalized, ok := normalizeJSON(response).(map[string]any)
	if !ok {
		t.Fatalf("response = %#v, want object", response)
	}
	return normalized, rec.Header().Get("X-Iroha-Cache")
}

func assertMetricMinor(t *testing.T, response map[string]any, want *int64) {
	t.Helper()
	series := response["series"].([]any)
	points := series[0].(map[string]any)["points"].([]any)
	point := points[0].(map[string]any)
	if want == nil {
		if point["value_minor"] != nil {
			t.Fatalf("metric point = %#v, want null value_minor", point)
		}
		return
	}
	value, ok := point["value_minor"].(float64)
	if !ok || int64(value) != *want {
		t.Fatalf("metric point = %#v, want value_minor %d", point, *want)
	}
}

func assertRouteCity(t *testing.T, response map[string]any, want string) {
	t.Helper()
	features := response["features"].([]any)
	properties := features[0].(map[string]any)["properties"].(map[string]any)
	if properties["city"] != want {
		t.Fatalf("route properties = %#v, want city %q", properties, want)
	}
}

func assertResolutionTaskStatus(t *testing.T, response map[string]any, want string) {
	t.Helper()
	items := response["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["status"] != want {
		t.Fatalf("resolution items = %#v, want one %q item", items, want)
	}
}

func int64Pointer(value int64) *int64 { return &value }
