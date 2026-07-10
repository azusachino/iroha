//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/config"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/imports"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/jobs"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/rawfiles"
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
	assertCanonicalImportedActivity(t, db, rawID, "Integration Run")

	var secondImportID string
	requestJSON(t, server, http.MethodPost, "/api/v1/imports", `{"raw_file_id":"`+rawID+`","parser_kind":"gpx"}`, http.StatusAccepted, func(body map[string]any) {
		secondImportID = stringValue(t, body, "id")
	})
	waitForImportStatus(t, server, secondImportID, imports.StatusCompleted)
	assertCanonicalImportedActivity(t, db, rawID, "Integration Run")

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

type testJobEnqueuer struct {
	jobsService *jobs.Service
}

func (e *testJobEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	return e.jobsService.EnqueueTx(tx, jobs.EnqueueInput{
		Kind:    kind,
		Payload: payload,
	})
}

func makeImportParseHandler(importService **imports.Service) jobs.Handler {
	return func(ctx context.Context, job models.Job) error {
		var payload struct {
			ImportJobID string `json:"import_job_id"`
		}
		if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
			return err
		}
		id, err := uuid.Parse(payload.ImportJobID)
		if err != nil {
			return err
		}
		if importService != nil && *importService != nil {
			return (*importService).Process(id)
		}
		return fmt.Errorf("import service not set")
	}
}

func newIntegrationServer(t *testing.T, db *gorm.DB) http.Handler {
	t.Helper()
	rawFileService, err := rawfiles.NewService(db, t.TempDir())
	if err != nil {
		t.Fatalf("new raw file service: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	var importService *imports.Service

	handlers := map[string]jobs.Handler{
		jobs.KindAppleImportParse:  makeImportParseHandler(&importService),
		jobs.KindGPXImportParse:    makeImportParseHandler(&importService),
		jobs.KindFITImportParse:    makeImportParseHandler(&importService),
		jobs.KindTCXImportParse:    makeImportParseHandler(&importService),
		jobs.KindStravaImportParse: makeImportParseHandler(&importService),
	}

	jobsService := jobs.NewService(db, logger, handlers)
	enqueuer := &testJobEnqueuer{jobsService: jobsService}
	importService = imports.NewService(db, logger, "integration-test", enqueuer, nil)

	// Start background test worker loop to process jobs from tb_jobs
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, err := jobsService.ProcessNext(ctx, "integration-test-worker")
				if err != nil && !errors.Is(err, jobs.ErrNoJobAvailable) {
					// ignore or log
				}
				time.Sleep(5 * time.Millisecond)
			}
		}
	}()

	return NewServer(Dependencies{
		Config: config.Config{
			Auth: config.AuthConfig{LocalNoAuth: true},
		},
		Logger:          logger,
		ActivityService: activities.NewService(db),
		ImportService:   importService,
		RawFileService:  rawFileService,
	})
}

func assertCanonicalImportedActivity(t *testing.T, db *gorm.DB, rawID string, wantTitle string) {
	t.Helper()
	decodedRawID, err := ids.Decode(ids.RawFilePrefix, rawID)
	if err != nil {
		t.Fatalf("decode raw id: %v", err)
	}

	assertTableCount(t, db, "tb_raw_files", 1)
	assertTableCount(t, db, "tb_activities", 1)
	assertTableCount(t, db, "tb_external_refs", 1)
	assertTableCount(t, db, "tb_activity_route_points", 2)
	assertTableCount(t, db, "tb_activity_samplings", 0)
	assertTableCount(t, db, "tb_activity_laps", 0)

	var row struct {
		ActivityID     uuid.UUID
		Title          string
		SportType      string
		SourceKind     string
		FirstRawFileID uuid.UUID
		RoutePoints    int
		FirstLon       float64
		FirstLat       float64
		LastLon        float64
		LastLat        float64
	}
	if err := db.Raw(`
		select
		  a.id as activity_id,
		  a.title,
		  a.sport_type,
		  a.source_kind,
		  a.first_raw_file_id,
		  count(rp.seq)::int as route_points,
		  min(ST_X(rp.geom::geometry)) filter (where rp.seq = 0) as first_lon,
		  min(ST_Y(rp.geom::geometry)) filter (where rp.seq = 0) as first_lat,
		  min(ST_X(rp.geom::geometry)) filter (where rp.seq = 1) as last_lon,
		  min(ST_Y(rp.geom::geometry)) filter (where rp.seq = 1) as last_lat
		from tb_activities a
		join tb_activity_route_points rp on rp.activity_id = a.id
		group by a.id
	`).Scan(&row).Error; err != nil {
		t.Fatalf("query canonical activity: %v", err)
	}
	if row.Title != wantTitle {
		t.Fatalf("title = %q, want %q", row.Title, wantTitle)
	}
	if row.SportType != "run" {
		t.Fatalf("sport_type = %q, want run", row.SportType)
	}
	if row.SourceKind != "gpx" {
		t.Fatalf("source_kind = %q, want gpx", row.SourceKind)
	}
	if row.FirstRawFileID != decodedRawID {
		t.Fatalf("first_raw_file_id = %s, want %s", row.FirstRawFileID, decodedRawID)
	}
	if row.RoutePoints != 2 {
		t.Fatalf("route point count = %d, want 2", row.RoutePoints)
	}
	if row.FirstLon != 139.0 || row.FirstLat != 35.0 || row.LastLon != 139.1 || row.LastLat != 35.1 {
		t.Fatalf("route geometry = (%v,%v)->(%v,%v), want (139,35)->(139.1,35.1)", row.FirstLon, row.FirstLat, row.LastLon, row.LastLat)
	}

	var refCount int64
	if err := db.Table("tb_external_refs").
		Where("activity_id = ? and provider = ? and raw_file_id = ?", row.ActivityID, "gpx", decodedRawID).
		Count(&refCount).Error; err != nil {
		t.Fatalf("query external refs: %v", err)
	}
	if refCount != 1 {
		t.Fatalf("external refs for imported activity = %d, want 1", refCount)
	}
}

func assertTableCount(t *testing.T, db *gorm.DB, table string, want int64) {
	t.Helper()
	var got int64
	if err := db.Table(table).Count(&got).Error; err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count = %d, want %d", table, got, want)
	}
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
