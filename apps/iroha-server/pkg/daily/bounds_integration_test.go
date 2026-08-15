//go:build integration

package daily

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

// TestServiceBoundsCapsMaxAtNowAndExtendsMinToOldestRow inserts one
// deliberately far-future and one deliberately far-past daily summary into
// the shared integration database (never truncated -- it holds real
// imported history) and checks Bounds against just those two rows,
// independent of whatever else the cross-domain union already contains.
func TestServiceBoundsCapsMaxAtNowAndExtendsMinToOldestRow(t *testing.T) {
	db := openDailyIntegrationDB(t)
	rawID := uuid.New()
	createdAt := time.Now().UTC()
	if err := db.Create(&models.RawFile{
		ID: rawID, SHA256: "daily-bounds-" + rawID.String(), OriginalFilename: "bounds.xml",
		StoragePath: "/tmp/bounds.xml", SourceKind: "test", UploadedVia: "test", CreatedAt: createdAt,
	}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	t.Cleanup(func() { db.Exec("delete from tb_raw_files where id = ?", rawID) })

	pastRow := models.DailySummary{
		ID: uuid.New(), Day: time.Date(1901, time.January, 2, 0, 0, 0, 0, time.UTC),
		Source: "test", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	futureRow := models.DailySummary{
		ID: uuid.New(), Day: time.Date(2150, time.June, 1, 0, 0, 0, 0, time.UTC),
		Source: "test", FirstRawFileID: rawID, CreatedAt: createdAt, UpdatedAt: createdAt,
	}
	for _, row := range []models.DailySummary{pastRow, futureRow} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create daily summary: %v", err)
		}
		t.Cleanup(func() { db.Exec("delete from tb_daily_summaries where id = ?", row.ID) })
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
