//go:build integration

package httpapi

import (
	"net/http"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

func TestIntegrationDateScopedCockpitUsesHalfOpenRangesAndAllDomainDates(t *testing.T) {
	db := openIntegrationDB(t)
	resetIntegrationDB(t, db)
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_media_consumption_events")
		_ = db.Exec("delete from tb_media_items")
		_ = db.Exec("delete from tb_media_works")
		resetIntegrationDB(t, db)
	})

	createdAt := time.Date(2099, time.January, 1, 0, 0, 0, 0, time.UTC)
	rawFile := models.RawFile{
		ID:               uuid.New(),
		SHA256:           "date-scope-" + uuid.NewString(),
		OriginalFilename: "date-scope.json",
		StoragePath:      "/tmp/date-scope.json",
		SourceKind:       "integration",
		UploadedVia:      "test",
		CreatedAt:        createdAt,
	}
	if err := db.Create(&rawFile).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}

	dailyDay := time.Date(2099, time.January, 2, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.DailySummary{
		ID: uuid.New(), Day: dailyDay, MoveKcal: 400, MoveGoalKcal: 300,
		ExerciseMin: 35, ExerciseGoalMin: 30, StandHours: 12, StandGoalHours: 10,
		Source: "integration", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create daily summary: %v", err)
	}

	sleepDay := time.Date(2099, time.January, 3, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.SleepSession{
		ID: uuid.New(), WakeDate: sleepDay,
		StartedAt:  time.Date(2099, time.January, 2, 22, 0, 0, 0, time.UTC),
		EndedAt:    time.Date(2099, time.January, 3, 6, 0, 0, 0, time.UTC),
		TimeInBedS: 28800, AsleepS: 25200, Efficiency: 0.875, IsMainSleep: true,
		Source: "integration", FirstRawFileID: rawFile.ID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create sleep session: %v", err)
	}

	activityDay := time.Date(2099, time.January, 4, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.Activity{
		ID: uuid.New(), SportType: "run", Title: "Date scope run", StartedAt: activityDay,
		Timezone: "UTC", SourceKind: "integration", FirstRawFileID: rawFile.ID,
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create activity: %v", err)
	}

	workID := uuid.New()
	itemID := uuid.New()
	if err := db.Create(&models.MediaWork{ID: workID, WorkKind: "book", PrimaryTitle: "Date scope book", CreatedAt: createdAt, UpdatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create media work: %v", err)
	}
	if err := db.Create(&models.MediaItem{ID: itemID, WorkID: &workID, MediaType: "book", ItemRole: "primary", Title: "Date scope book", CreatedAt: createdAt, UpdatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create media item: %v", err)
	}
	mediaDay := time.Date(2099, time.January, 5, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&models.MediaConsumptionEvent{
		ID: uuid.New(), MediaItemID: itemID, EventType: "finished", EventAt: &mediaDay,
		SourceKind: "integration", CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create media event: %v", err)
	}

	server := newIntegrationServer(t, db)
	briefing := requestJSON(t, server, http.MethodGet, "/api/v1/briefing?date=2099-01-02", "", http.StatusOK, nil)
	if section := integrationBriefingSection(t, briefing, "daily"); section["state"] != "ready" {
		t.Fatalf("daily briefing section = %#v, want ready", section)
	}
	sleepBriefing := requestJSON(t, server, http.MethodGet, "/api/v1/briefing?date=2099-01-03", "", http.StatusOK, nil)
	if section := integrationBriefingSection(t, sleepBriefing, "sleep"); section["state"] != "ready" {
		t.Fatalf("sleep briefing section = %#v, want ready", section)
	}

	dates := requestJSON(t, server, http.MethodGet, "/api/v1/daily/dates", "", http.StatusOK, nil)["items"].([]any)
	got := make(map[string]bool, len(dates))
	for _, value := range dates {
		got[value.(string)] = true
	}
	for _, want := range []string{"2099-01-02", "2099-01-03", "2099-01-04", "2099-01-05"} {
		if !got[want] {
			t.Errorf("daily date index missing %s: %v", want, got)
		}
	}
}

func integrationBriefingSection(t *testing.T, response map[string]any, key string) map[string]any {
	t.Helper()
	sections, ok := response["sections"].([]any)
	if !ok {
		t.Fatalf("sections = %#v", response["sections"])
	}
	for _, value := range sections {
		section := value.(map[string]any)
		if section["key"] == key {
			return section
		}
	}
	t.Fatalf("missing briefing section %q: %#v", key, sections)
	return nil
}
