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

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/mediaresolution"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metricseries"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
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
	requestJSON(t, server, http.MethodGet, "/api/v1/raw-files/raw_bad", "", http.StatusBadRequest, assertErrorResponse(t, "invalid_raw_file_id", "invalid raw_file id"))
	requestJSON(t, server, http.MethodGet, "/api/v1/raw-files/"+ids.Encode(ids.RawFilePrefix, uuid.New()), "", http.StatusNotFound, assertErrorResponse(t, "raw_file_not_found", "raw_file not found"))

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
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/summary", "", http.StatusOK, func(body map[string]any) {
		totals := body["totals"].(map[string]any)
		if totals["distance_known_count"] != float64(0) || totals["distance_unknown_count"] != float64(1) || totals["moving_time_s"] != nil {
			t.Fatalf("activity summary totals = %#v", totals)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/activities/summary?date=2099-12-31", "", http.StatusBadRequest, nil)
}

func TestIntegrationSleepEndpoints(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })

	rawFile := models.RawFile{
		ID:               uuid.New(),
		SHA256:           "sleep-integration",
		OriginalFilename: "sleep.xml",
		StoragePath:      "/tmp/sleep.xml",
		SourceKind:       "apple_health_export",
		UploadedVia:      "test",
		CreatedAt:        time.Now().UTC(),
	}
	if err := db.Create(&rawFile).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}

	firstID := uuid.New()
	secondID := uuid.New()
	napID := uuid.New()
	createdAt := time.Now().UTC()
	for _, session := range []models.SleepSession{
		{
			ID: firstID, WakeDate: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
			StartedAt: time.Date(2024, time.January, 1, 22, 0, 0, 0, time.UTC), EndedAt: time.Date(2024, time.January, 2, 6, 0, 0, 0, time.UTC),
			TimeInBedS: 28800, AsleepS: 25200, Efficiency: 0.875, IsMainSleep: true,
			CoreS: 12000, DeepS: 6000, RemS: 7200, AwakeS: 3600, Source: "Watch", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt,
		},
		{
			ID: secondID, WakeDate: time.Date(2023, time.December, 31, 0, 0, 0, 0, time.UTC),
			StartedAt: time.Date(2023, time.December, 30, 23, 0, 0, 0, time.UTC), EndedAt: time.Date(2023, time.December, 31, 7, 0, 0, 0, time.UTC),
			TimeInBedS: 28800, AsleepS: 21600, Efficiency: 0.75, IsMainSleep: true,
			CoreS: 12000, DeepS: 3600, RemS: 6000, AwakeS: 7200, Source: "Watch", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt,
		},
		{
			ID: napID, WakeDate: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			StartedAt: time.Date(2024, time.January, 1, 13, 0, 0, 0, time.UTC), EndedAt: time.Date(2024, time.January, 1, 14, 0, 0, 0, time.UTC),
			TimeInBedS: 3600, AsleepS: 1800, Efficiency: 0.5, IsMainSleep: false,
			CoreS: 900, DeepS: 300, RemS: 600, AwakeS: 1800, Source: "Watch", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt,
		},
	} {
		if err := db.Create(&session).Error; err != nil {
			t.Fatalf("create sleep session: %v", err)
		}
	}
	segmentID := uuid.New()
	if err := db.Create(&models.SleepSegment{
		ID: segmentID, SessionID: firstID, Stage: "core", StartedAt: time.Date(2024, time.January, 1, 23, 0, 0, 0, time.UTC), EndedAt: time.Date(2024, time.January, 2, 1, 0, 0, 0, time.UTC), Seq: 0,
	}).Error; err != nil {
		t.Fatalf("create sleep segment: %v", err)
	}

	server := newIntegrationServer(t, db)
	firstPage := requestJSON(t, server, http.MethodGet, "/api/v1/sleep/?limit=1", "", http.StatusOK, func(body map[string]any) {
		if len(body["items"].([]any)) != 1 || body["has_more"] != true {
			t.Fatalf("first sleep page = %#v", body)
		}
		item := body["items"].([]any)[0].(map[string]any)
		startedAt, err := time.Parse(time.RFC3339, item["started_at"].(string))
		if err != nil || !startedAt.Equal(time.Date(2024, time.January, 1, 22, 0, 0, 0, time.UTC)) || item["wake_date"] != "2024-01-02" {
			t.Fatalf("sleep wire dates = %#v", item)
		}
	})
	cursor := stringValue(t, firstPage, "next_cursor")
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/?limit=1&cursor="+cursor, "", http.StatusOK, func(body map[string]any) {
		items := body["items"].([]any)
		if len(items) != 1 || body["has_more"] != true {
			t.Fatalf("second sleep page = %#v", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/?from=2024-01-02&to=2024-01-03", "", http.StatusOK, func(body map[string]any) {
		if len(body["items"].([]any)) != 1 {
			t.Fatalf("date-filtered sleep page = %#v", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/"+ids.Encode(ids.SleepPrefix, firstID), "", http.StatusOK, func(body map[string]any) {
		if body["id"] != ids.Encode(ids.SleepPrefix, firstID) {
			t.Fatalf("sleep detail = %#v", body)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/aggregates?granularity=year", "", http.StatusOK, func(body map[string]any) {
		buckets := body["buckets"].([]any)
		if body["granularity"] != "year" || len(buckets) != 2 {
			t.Fatalf("yearly sleep aggregates = %#v", body)
		}
		if buckets[0].(map[string]any)["period"] != "2023" || buckets[0].(map[string]any)["session_count"] != float64(1) {
			t.Fatalf("yearly aggregate bucket = %#v", buckets[0])
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/aggregates?granularity=month&from=2024-01-01&to=2024-01-31", "", http.StatusOK, func(body map[string]any) {
		buckets := body["buckets"].([]any)
		if len(buckets) != 1 {
			t.Fatalf("monthly sleep aggregates = %#v", body)
		}
		bucket := buckets[0].(map[string]any)
		if bucket["period"] != "2024-01" || bucket["session_count"] != float64(2) || bucket["main_sleep_count"] != float64(1) || bucket["nap_count"] != float64(1) || bucket["observed_wake_dates"] != float64(2) || bucket["average_asleep_s"] != float64(25200) || bucket["core_s"] != float64(12000) {
			t.Fatalf("monthly sleep aggregate bucket = %#v", bucket)
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/aggregates?granularity=week", "", http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/sleep_bad/segments", "", http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/sleep/"+ids.Encode(ids.SleepPrefix, firstID)+"/segments", "", http.StatusOK, func(body map[string]any) {
		items := body["items"].([]any)
		if len(items) != 1 || items[0].(map[string]any)["id"] != ids.Encode(ids.SleepSegmentPrefix, segmentID) {
			t.Fatalf("sleep segments = %#v", body)
		}
	})
}

func TestIntegrationDailyEndpoint(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })

	rawFile := models.RawFile{
		ID:               uuid.New(),
		SHA256:           "daily-integration",
		OriginalFilename: "daily.xml",
		StoragePath:      "/tmp/daily.xml",
		SourceKind:       "apple_health_export",
		UploadedVia:      "test",
		CreatedAt:        time.Now().UTC(),
	}
	if err := db.Create(&rawFile).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	createdAt := time.Now().UTC()
	for _, summary := range []models.DailySummary{
		{ID: uuid.New(), Day: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), MoveKcal: 600, MoveGoalKcal: 500, ExerciseMin: 45, ExerciseGoalMin: 30, StandHours: 10, StandGoalHours: 12, Source: "apple_health_export", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), Day: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC), MoveKcal: 400, MoveGoalKcal: 500, ExerciseMin: 20, ExerciseGoalMin: 30, StandHours: 8, StandGoalHours: 12, Source: "apple_health_export", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt},
	} {
		if err := db.Create(&summary).Error; err != nil {
			t.Fatalf("create daily summary: %v", err)
		}
		if err := db.Create(&models.DailyMetric{ID: uuid.New(), Day: summary.Day, Metric: "steps", Value: 1234, Unit: "count", Source: "Watch", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt}).Error; err != nil {
			t.Fatalf("create daily metric: %v", err)
		}
	}
	for _, metric := range []models.DailyMetric{
		{ID: uuid.New(), Day: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), Metric: "resting_hr", Value: 57, Unit: "count/min", Source: "Watch", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), Day: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC), Metric: "hrv_sdnn", Value: 42, Unit: "ms", Source: "Watch", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt},
		{ID: uuid.New(), Day: time.Date(2023, time.December, 31, 0, 0, 0, 0, time.UTC), Metric: "body_mass_kg", Value: 70.5, Unit: "kg", Source: "Watch", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt},
	} {
		if err := db.Create(&metric).Error; err != nil {
			t.Fatalf("create vitals metric: %v", err)
		}
	}

	server := newIntegrationServer(t, db)
	firstPage := requestJSON(t, server, http.MethodGet, "/api/v1/daily?limit=1", "", http.StatusOK, func(body map[string]any) {
		items := body["items"].([]any)
		if len(items) != 1 || body["has_more"] != true {
			t.Fatalf("first daily page = %#v", body)
		}
		item := items[0].(map[string]any)
		if item["day"] != "2024-01-02" || item["steps"] != float64(1234) || item["resting_hr"] != float64(57) || item["hrv_sdnn"] != float64(42) {
			t.Fatalf("daily item = %#v", item)
		}
		if ring := item["ring"].(map[string]any); ring["move_kcal"] != float64(600) {
			t.Fatalf("daily ring = %#v", ring)
		}
	})
	cursor := stringValue(t, firstPage, "next_cursor")
	secondPage := requestJSON(t, server, http.MethodGet, "/api/v1/daily?limit=1&cursor="+cursor, "", http.StatusOK, func(body map[string]any) {
		items := body["items"].([]any)
		if len(items) != 1 || body["has_more"] != true {
			t.Fatalf("second daily page = %#v", body)
		}
		if items[0].(map[string]any)["day"] != "2024-01-01" {
			t.Fatalf("second daily item = %#v", items[0])
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/daily?limit=1&cursor="+stringValue(t, secondPage, "next_cursor"), "", http.StatusOK, func(body map[string]any) {
		items := body["items"].([]any)
		if len(items) != 1 || body["has_more"] != false {
			t.Fatalf("third daily page = %#v", body)
		}
		item := items[0].(map[string]any)
		if item["day"] != "2023-12-31" || item["body_mass_kg"] != float64(70.5) {
			t.Fatalf("metric-only daily item = %#v", item)
		}
		if item["ring"] != nil {
			t.Fatalf("metric-only daily ring = %#v, want null", item["ring"])
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/daily/aggregates?granularity=month&from=2024-01-01&to=2024-01-31", "", http.StatusOK, func(body map[string]any) {
		buckets := body["buckets"].([]any)
		if len(buckets) != 1 {
			t.Fatalf("daily aggregates = %#v", body)
		}
		bucket := buckets[0].(map[string]any)
		metrics := bucket["metrics"].([]any)
		if bucket["period"] != "2024-01" || len(metrics) != 3 {
			t.Fatalf("daily aggregate bucket = %#v", bucket)
		}
		for _, value := range metrics {
			metric := value.(map[string]any)
			if metric["metric"] == "steps" && (metric["unit"] != "count" || metric["observed_days"] != float64(2)) {
				t.Fatalf("steps aggregate = %#v", metric)
			}
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/daily?from=bad", "", http.StatusBadRequest, nil)
}

func TestIntegrationMonthlyReportEndpoint(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() { resetIntegrationDB(t, db) })
	server := newIntegrationServer(t, db)

	requestJSON(t, server, http.MethodGet, "/api/v1/reports/monthly?month=2026-08&timezone=UTC", "", http.StatusOK, func(body map[string]any) {
		if body["schema"] != "monthly-report.v1" {
			t.Fatalf("schema = %#v", body["schema"])
		}
		period := body["period"].(map[string]any)
		if period["month"] != "2026-08" || period["from"] != "2026-08-01" || period["to"] != "2026-09-01" || period["timezone"] != "UTC" {
			t.Fatalf("period = %#v", period)
		}
		sections := body["sections"].(map[string]any)
		for _, name := range []string{"movement", "sleep", "daily_health", "media", "expenses"} {
			section := sections[name].(map[string]any)
			state := section["state"].(string)
			if state != "empty" && state != "available" {
				t.Fatalf("section %s = %#v", name, section)
			}
			if state == "empty" && section["data"] != nil {
				t.Fatalf("empty section %s has data: %#v", name, section)
			}
			if state == "available" && section["data"] == nil {
				t.Fatalf("available section %s has no data: %#v", name, section)
			}
		}
	})
	requestJSON(t, server, http.MethodGet, "/api/v1/reports/monthly?month=2026-13", "", http.StatusBadRequest, nil)
	requestJSON(t, server, http.MethodGet, "/api/v1/reports/monthly?month=2026-08&timezone=Not%2FATimezone", "", http.StatusBadRequest, nil)
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
		tb_expenses,
		tb_geocode_cache,
		tb_activity_laps,
		tb_activity_samplings,
		tb_activity_route_points,
		 tb_external_refs,
		 tb_activities,
		 tb_media_resolution_tasks,
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
	return newIntegrationServerWithCache(t, db, nil)
}

func newIntegrationServerWithCache(t *testing.T, db *gorm.DB, responseCache *cache.Client) http.Handler {
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
	importService = imports.NewService(db, logger, "integration-test", enqueuer, responseCache)
	geocodeService := geocode.NewService(db, enqueuer, responseCache)
	metricRegistry, err := metrics.DefaultRegistry()
	if err != nil {
		t.Fatalf("create metric registry: %v", err)
	}
	metricSeriesService := metricseries.NewService(
		metricRegistry,
		metricseries.DailyServiceSource{Service: daily.NewService(db)},
		metricseries.ActivityServiceSource{Service: activities.NewService(db)},
		metricseries.ExpenseServiceSource{Service: expenses.NewService(db)},
		metricseries.SleepServiceSource{Service: sleep.NewService(db)},
		metricseries.MediaServiceSource{Service: media.NewService(db)},
	)
	dailyService := daily.NewService(db)
	sleepService := sleep.NewService(db)
	activityService := activities.NewService(db)
	mediaService := media.NewService(db)
	briefingRegistry, err := NewBriefingRegistry(dailyService, sleepService, activityService, mediaService)
	if err != nil {
		t.Fatalf("create briefing registry: %v", err)
	}

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
		Config:                 config.Config{},
		Now:                    func() time.Time { return time.Date(2099, time.December, 31, 12, 0, 0, 0, time.UTC) },
		Logger:                 logger,
		ActivityService:        activityService,
		SleepService:           sleepService,
		DailyService:           dailyService,
		ExpenseService:         expenses.NewService(db),
		MediaService:           mediaService,
		MetricRegistry:         metricRegistry,
		MetricSeriesService:    metricSeriesService,
		BriefingRegistry:       briefingRegistry,
		MediaResolutionService: mediaresolution.NewService(db),
		ImportService:          importService,
		RawFileService:         rawFileService,
		Cache:                  responseCache,
		GeocodeService:         geocodeService,
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

func assertErrorResponse(t *testing.T, wantCode, wantMessage string) func(map[string]any) {
	t.Helper()
	return func(body map[string]any) {
		if body["code"] != wantCode || body["message"] != wantMessage {
			t.Errorf("error body = %#v, want code=%q message=%q", body, wantCode, wantMessage)
		}
		if requestID, ok := body["request_id"].(string); !ok || requestID == "" {
			t.Errorf("error body has no request_id: %#v", body)
		}
	}
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
