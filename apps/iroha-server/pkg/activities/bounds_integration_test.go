//go:build integration

package activities

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

// TestServiceBoundsCapsMaxAtNowAndExtendsMinToOldestRow inserts one
// deliberately far-future and one deliberately far-past activity into the
// shared integration database (never truncated -- it holds real imported
// history) and checks Bounds against just those two rows, independent of
// whatever else exists.
func TestServiceBoundsCapsMaxAtNowAndExtendsMinToOldestRow(t *testing.T) {
	db := openActivitiesIntegrationDB(t)
	rawID := uuid.New()
	createdAt := time.Now().UTC()
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: "activities-bounds-" + rawID.String(), OriginalFilename: "bounds.gpx",
		StoragePath: "/tmp/bounds.gpx", SourceKind: "gpx", UploadedVia: "test", CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() { db.Exec("delete from tb_raw_files where id = ?", rawID) })

	pastRow := models.Activity{
		ID: uuid.New(), SportType: "run",
		StartedAt:  time.Date(1901, time.January, 2, 0, 0, 0, 0, time.UTC),
		SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	futureRow := models.Activity{
		ID: uuid.New(), SportType: "run",
		StartedAt:  time.Date(2150, time.June, 1, 0, 0, 0, 0, time.UTC),
		SourceKind: "gpx", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	for _, row := range []models.Activity{pastRow, futureRow} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create activity: %v", err)
		}
		t.Cleanup(func() { db.Exec("delete from tb_activities where id = ?", row.ID) })
	}

	now := time.Date(2026, time.August, 15, 12, 0, 0, 0, time.UTC)
	minDate, maxDate, ok, err := NewService(db).Bounds(now, "UTC")
	if err != nil {
		t.Fatalf("bounds: %v", err)
	}
	if !ok {
		t.Fatal("bounds ok = false, want true")
	}
	if minDate != "1901-01-02" {
		t.Fatalf("min = %q, want 1901-01-02 (the oldest row inserted by this test)", minDate)
	}
	if maxDate != "2026-08-15" {
		t.Fatalf("max = %q, want 2026-08-15 (capped at now, not the 2150 row)", maxDate)
	}
}
