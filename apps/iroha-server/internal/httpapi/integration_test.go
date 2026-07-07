//go:build integration

package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/internal/activities"
	"github.com/azusachino/iroha/apps/iroha-server/internal/config"
	"github.com/azusachino/iroha/apps/iroha-server/internal/ids"
	"github.com/azusachino/iroha/apps/iroha-server/internal/imports"
	"github.com/azusachino/iroha/apps/iroha-server/internal/rawfiles"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestIntegrationRawFileImportAndActivityEndpoints(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })

	server := newIntegrationServer(t, db)

	rawID := uploadRawFile(t, server, "run.gpx", "gpx", "cli", validGPX())
	duplicateID := uploadRawFile(t, server, "copy.gpx", "gpx", "web", validGPX())
	if duplicateID != rawID {
		t.Fatalf("duplicate raw id = %q, want %q", duplicateID, rawID)
	}

	requestJSON(t, server, http.MethodGet, "/api/v1/raw-files/"+rawID, "", http.StatusOK, func(body map[string]any) {
		if body["duplicate"] != nil {
			t.Fatalf("duplicate marker leaked into get response: %#v", body)
		}
		if body["uploaded_via"] != "cli" {
			t.Fatalf("uploaded_via = %v, want cli", body["uploaded_via"])
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/raw-files/raw_bad", "", http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/raw-files/"+ids.Encode(ids.RawFilePrefix, uuid.New()), "", http.StatusNotFound, nil)

	var importID string
	requestJSON(t, server, http.MethodPost, "/api/v1/imports", `{"raw_file_id":"`+rawID+`","parser_kind":"gpx"}`, http.StatusAccepted, func(body map[string]any) {
		importID = stringValue(t, body, "id")
		if body["raw_file_id"] != rawID {
			t.Fatalf("raw_file_id = %v, want %s", body["raw_file_id"], rawID)
		}
	})
	waitForImportStatus(t, server, importID, imports.StatusCompleted)
	requestJSON(t, server, http.MethodPost, "/api/v1/imports", `{"raw_file_id":"`+rawID+`","parser_kind":"bogus"}`, http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/imports/imp_bad", "", http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/imports/"+ids.Encode(ids.ImportPrefix, uuid.New()), "", http.StatusNotFound, nil)

	activityID := firstActivityID(t, server)
	requestJSON(t, server, http.MethodGet, "/api/v1/activities?limit=bad", "", http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/activities?sport_type=ride", "", http.StatusOK, func(body map[string]any) {
		if len(body["items"].([]any)) != 0 {
			t.Fatalf("ride filter returned rows: %#v", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/"+activityID, "", http.StatusOK, func(body map[string]any) {
		if body["sport_type"] != "run" {
			t.Fatalf("sport_type = %v, want run", body["sport_type"])
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/act_bad", "", http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/"+ids.Encode(ids.ActivityPrefix, uuid.New()), "", http.StatusNotFound, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/"+activityID+"/route", "", http.StatusOK, func(body map[string]any) {
		if len(body["items"].([]any)) != 2 {
			t.Fatalf("route points = %#v, want 2", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/"+activityID+"/samplings", "", http.StatusOK, func(body map[string]any) {
		if len(body["items"].([]any)) != 0 {
			t.Fatalf("samplings = %#v, want none", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/"+activityID+"/laps", "", http.StatusOK, func(body map[string]any) {
		if len(body["items"].([]any)) != 0 {
			t.Fatalf("laps = %#v, want none", body)
		}
	})
}

func openIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://iroha:iroha_dev@127.0.0.1:5432/iroha?sslmode=disable"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func resetIntegrationDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(`truncate table
		tb_activity_laps,
		tb_activity_samplings,
		tb_activity_route_points,
		tb_external_refs,
		tb_activities,
		tb_import_jobs,
		tb_raw_files
		cascade`).Error; err != nil {
		t.Fatalf("reset integration db: %v", err)
	}
}

func newIntegrationServer(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	rawFileService, err := rawfiles.NewService(db, t.TempDir())
	if err != nil {
		t.Fatalf("new raw file service: %v", err)
	}
	return NewServer(Dependencies{
		Config: config.Config{
			Auth: config.AuthConfig{LocalNoAuth: true},
		},
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		ActivityService: activities.NewService(db),
		ImportService:   imports.NewService(db, slog.New(slog.NewTextHandler(io.Discard, nil)), "integration-test"),
		RawFileService:  rawFileService,
	})
}

func uploadRawFile(t *testing.T, handler http.Handler, filename string, sourceKind string, uploadedVia string, content string) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("source_kind", sourceKind); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("uploaded_via", uploadedVia); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/raw-files/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("upload status = %d body = %s", rec.Code, rec.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return stringValue(t, response, "id")
}

func requestJSON(t *testing.T, handler http.Handler, method string, path string, body string, wantStatus int, check func(map[string]any)) map[string]any {
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
	normalized := normalizeJSON(response).(map[string]any)
	if check != nil {
		check(normalized)
	}
	return normalized
}

func normalizeJSON(value any) any {
	switch typed := value.(type) {
	case []any:
		return map[string]any{"items": typed}
	case map[string]any:
		return typed
	default:
		return value
	}
}

func waitForImportStatus(t *testing.T, handler http.Handler, importID string, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		response := requestJSON(t, handler, http.MethodGet, "/api/v1/imports/"+importID, "", http.StatusOK, nil)
		if response["status"] == want {
			return
		}
		if response["status"] == imports.StatusFailed {
			t.Fatalf("import failed: %#v", response)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("import %s did not reach %s", importID, want)
}

func firstActivityID(t *testing.T, handler http.Handler) string {
	t.Helper()
	response := requestJSON(t, handler, http.MethodGet, "/api/v1/activities?sport_type=run", "", http.StatusOK, nil)
	items := response["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("activities = %#v, want one", items)
	}
	item := items[0].(map[string]any)
	return stringValue(t, item, "id")
}

func stringValue(t *testing.T, values map[string]any, key string) string {
	t.Helper()
	value, ok := values[key].(string)
	if !ok || value == "" {
		t.Fatalf("%s = %#v, want non-empty string", key, values[key])
	}
	return value
}

func validGPX() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="iroha-integration-test">
  <trk>
    <name>Integration Run</name>
    <trkseg>
      <trkpt lat="35.0" lon="139.0">
        <ele>12.5</ele>
        <time>2026-07-07T00:00:00Z</time>
      </trkpt>
      <trkpt lat="35.1" lon="139.1">
        <ele>13.5</ele>
        <time>2026-07-07T00:05:00Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
}
