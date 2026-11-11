//go:build integration

package main

import (
	"testing"

	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/testdb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestIntegrationConfiguredMediaScheduleIsIdempotentAndOptOutable(t *testing.T) {
	dsn := testdb.DSN(t)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get integration db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncAniList, true, "12h"); err != nil {
		t.Fatalf("enable AniList schedule: %v", err)
	}
	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncAniList, true, "12h"); err != nil {
		t.Fatalf("reconcile AniList schedule: %v", err)
	}
	var schedule models.JobSchedule
	if err := db.Where("kind = ?", jobs.KindMediaSyncAniList).First(&schedule).Error; err != nil {
		t.Fatalf("load AniList schedule: %v", err)
	}
	if !schedule.Enabled || schedule.ScheduleExpr != "12h0m0s" {
		t.Fatalf("schedule = %#v, want enabled 12h0m0s", schedule)
	}
	var count int64
	if err := db.Model(&models.JobSchedule{}).Where("kind = ?", jobs.KindMediaSyncAniList).Count(&count).Error; err != nil {
		t.Fatalf("count AniList schedules: %v", err)
	}
	if count != 1 {
		t.Fatalf("AniList schedules = %d, want 1", count)
	}

	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncAniList, true, "off"); err != nil {
		t.Fatalf("opt out AniList schedule: %v", err)
	}
	if err := db.First(&schedule, "kind = ?", jobs.KindMediaSyncAniList).Error; err != nil {
		t.Fatalf("reload opted-out schedule: %v", err)
	}
	if schedule.Enabled {
		t.Fatal("opted-out schedule remained enabled")
	}
	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncBangumi, false, ""); err != nil {
		t.Fatalf("skip unconfigured Bangumi schedule: %v", err)
	}
	if err := db.Model(&models.JobSchedule{}).Where("kind = ?", jobs.KindMediaSyncBangumi).Count(&count).Error; err != nil {
		t.Fatalf("count Bangumi schedules: %v", err)
	}
	if count != 0 {
		t.Fatalf("unconfigured Bangumi schedules = %d, want 0", count)
	}
}

// TestIntegrationConfiguredDrifterScheduleReconciles covers the Drifter
// gateway's schedule the same way: an API token alone does not schedule a sync,
// IROHA_DRIFTER_SYNC_INTERVAL sets its cadence, and "off" disables it.
func TestIntegrationConfiguredDrifterScheduleReconciles(t *testing.T) {
	dsn := testdb.DSN(t)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get integration db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	// A previous run may have left this kind behind; the count assertions below
	// describe this test's own reconciliations, not the database's history.
	if err := db.Where("kind = ?", jobs.KindMediaSyncDrifter).Delete(&models.JobSchedule{}).Error; err != nil {
		t.Fatalf("clear Drifter schedules: %v", err)
	}

	var count int64
	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncDrifter, false, "6h"); err != nil {
		t.Fatalf("skip unconfigured Drifter schedule: %v", err)
	}
	if err := db.Model(&models.JobSchedule{}).Where("kind = ?", jobs.KindMediaSyncDrifter).Count(&count).Error; err != nil {
		t.Fatalf("count Drifter schedules: %v", err)
	}
	if count != 0 {
		t.Fatalf("unconfigured Drifter schedules = %d, want 0", count)
	}

	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncDrifter, true, "6h"); err != nil {
		t.Fatalf("enable Drifter schedule: %v", err)
	}
	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncDrifter, true, "6h"); err != nil {
		t.Fatalf("reconcile Drifter schedule: %v", err)
	}
	var schedule models.JobSchedule
	if err := db.Where("kind = ?", jobs.KindMediaSyncDrifter).First(&schedule).Error; err != nil {
		t.Fatalf("load Drifter schedule: %v", err)
	}
	if !schedule.Enabled || schedule.ScheduleExpr != "6h0m0s" {
		t.Fatalf("schedule = %#v, want enabled 6h0m0s", schedule)
	}
	if err := db.Model(&models.JobSchedule{}).Where("kind = ?", jobs.KindMediaSyncDrifter).Count(&count).Error; err != nil {
		t.Fatalf("count Drifter schedules: %v", err)
	}
	if count != 1 {
		t.Fatalf("Drifter schedules = %d, want 1", count)
	}

	if err := ensureConfiguredMediaSchedule(db, jobs.KindMediaSyncDrifter, true, "off"); err != nil {
		t.Fatalf("opt out Drifter schedule: %v", err)
	}
	if err := db.First(&schedule, "kind = ?", jobs.KindMediaSyncDrifter).Error; err != nil {
		t.Fatalf("reload opted-out Drifter schedule: %v", err)
	}
	if schedule.Enabled {
		t.Fatal("opted-out Drifter schedule remained enabled")
	}
}
