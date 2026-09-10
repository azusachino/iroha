//go:build integration

package imports

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	providerregistry "github.com/azusachino/iroha/apps/iroha-providers/registry"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestIntegrationAppleHealthShortcutReplayAndPartialWindow(t *testing.T) {
	db := openImportsIntegrationDB(t)
	fixturePath := filepath.Join("..", "iroha-providers", "parsers", "testdata", "apple_health_shortcut.json")
	body, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read shortcut fixture: %v", err)
	}
	partialBody := []byte(`{
  "schema": "iroha.health.shortcut.v1",
  "source_instance_key": "iphone-health:primary",
  "captured_at": "2026-09-11T23:30:00+09:00",
  "coverage": [{"category":"activities","scope":{"workout_types":["running"]},"from":"2026-09-04T00:00:00+09:00","to":"2026-09-11T00:00:00+09:00","timezone":"Asia/Tokyo","completeness":"partial"}],
  "activities": [],
  "sleep": [],
  "daily_summaries": [],
  "daily_metrics": []
}`)

	instanceID := uuid.New()
	now := time.Now().UTC()
	sourceKey := "iphone-health:primary:" + instanceID.String()
	body = bytes.Replace(body, []byte("iphone-health:primary"), []byte(sourceKey), 1)
	partialBody = bytes.Replace(partialBody, []byte("iphone-health:primary"), []byte(sourceKey), 1)
	queueIDs := make([]uuid.UUID, 0, 3)
	if err := db.Create(&models.SourceInstance{ID: instanceID, Provider: "apple_health", InstanceKey: sourceKey, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create source instance: %v", err)
	}
	cleanup := make([]uuid.UUID, 0, 2)
	t.Cleanup(func() {
		for _, queueID := range queueIDs {
			_ = db.Exec("delete from tb_jobs where id = ?", queueID).Error
		}
		_ = db.Exec("delete from tb_source_coverage_assertions where source_instance_id = ?", instanceID).Error
		_ = db.Exec("delete from tb_source_observation_receipts where source_instance_id = ?", instanceID).Error
		_ = db.Exec("update tb_activities set selected_observation_id = null where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("update tb_sleep_sessions set selected_observation_id = null where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("update tb_daily_summaries set selected_observation_id = null where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("update tb_daily_metrics set selected_observation_id = null where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("delete from tb_activities where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("delete from tb_sleep_sessions where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("delete from tb_daily_summaries where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("delete from tb_daily_metrics where first_raw_file_id in ?", cleanup).Error
		_ = db.Exec("delete from tb_source_observations where source_instance_id = ?", instanceID).Error
		_ = db.Exec("delete from tb_import_snapshots where raw_file_id in ?", cleanup).Error
		_ = db.Exec("delete from tb_import_jobs where raw_file_id in ?", cleanup).Error
		_ = db.Exec("delete from tb_source_receipts where source_instance_id = ?", instanceID).Error
		_ = db.Exec("delete from tb_raw_files where id in ?", cleanup).Error
		_ = db.Exec("delete from tb_source_instances where id = ?", instanceID).Error
	})

	service := NewServiceWithRegistry(db, nil, DefaultParserVersion, nil, nil, mustProviderRegistry(t))
	firstRawID, firstImportID, firstQueueID := createShortcutEvidence(t, db, body, instanceID, now)
	cleanup = append(cleanup, firstRawID)
	queueIDs = append(queueIDs, firstQueueID)
	processShortcutJob(t, service, db, firstImportID, firstQueueID)

	var observationCount int64
	if err := db.Model(&models.SourceObservation{}).Where("source_instance_id = ? and source_kind = ?", instanceID, "activity").Count(&observationCount).Error; err != nil {
		t.Fatalf("count first observation: %v", err)
	}
	if observationCount != 1 {
		t.Fatalf("activity observations after first import = %d, want 1", observationCount)
	}
	var coverageCount int64
	if err := db.Model(&models.SourceCoverageAssertion{}).Where("source_instance_id = ?", instanceID).Count(&coverageCount).Error; err != nil {
		t.Fatalf("count first coverage: %v", err)
	}
	if coverageCount != 3 {
		t.Fatalf("coverage assertions after first import = %d, want 3", coverageCount)
	}

	duplicateImportID, duplicateQueueID := createImportReplay(t, db, firstRawID, coreimports.KindAppleHealthShortcut, now.Add(time.Minute))
	queueIDs = append(queueIDs, duplicateQueueID)
	processShortcutJob(t, service, db, duplicateImportID, duplicateQueueID)
	if err := db.Model(&models.SourceCoverageAssertion{}).Where("source_instance_id = ?", instanceID).Count(&coverageCount).Error; err != nil {
		t.Fatalf("count duplicate coverage: %v", err)
	}
	if coverageCount != 3 {
		t.Fatalf("coverage assertions after duplicate replay = %d, want 3", coverageCount)
	}

	partialRawID, partialImportID, partialQueueID := createShortcutEvidence(t, db, partialBody, instanceID, now.Add(2*time.Minute))
	cleanup = append(cleanup, partialRawID)
	queueIDs = append(queueIDs, partialQueueID)
	processShortcutJob(t, service, db, partialImportID, partialQueueID)
	if err := db.Model(&models.SourceObservation{}).Where("source_instance_id = ? and source_kind = ?", instanceID, "activity").Count(&observationCount).Error; err != nil {
		t.Fatalf("count partial observation: %v", err)
	}
	if observationCount != 1 {
		t.Fatalf("activity observations after partial import = %d, want 1", observationCount)
	}
	if err := db.Model(&models.SourceCoverageAssertion{}).Where("source_instance_id = ?", instanceID).Count(&coverageCount).Error; err != nil {
		t.Fatalf("count partial coverage: %v", err)
	}
	if coverageCount != 4 {
		t.Fatalf("coverage assertions after partial import = %d, want 4", coverageCount)
	}
}

func mustProviderRegistry(t *testing.T) *provider.Registry {
	t.Helper()
	registry, err := providerregistry.New()
	if err != nil {
		t.Fatalf("build provider registry: %v", err)
	}
	return registry
}

func createShortcutEvidence(t *testing.T, db *gorm.DB, body []byte, instanceID uuid.UUID, observedAt time.Time) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	rawID := uuid.New()
	hash := sha256.Sum256(body)
	storagePath := filepath.Join(t.TempDir(), "shortcut.json")
	if err := os.WriteFile(storagePath, body, 0o600); err != nil {
		t.Fatalf("write shortcut body: %v", err)
	}
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: hex.EncodeToString(hash[:]), OriginalFilename: "apple-health-shortcut.json", ContentType: "application/json", SizeBytes: int64(len(body)), StoragePath: storagePath, SourceKind: coreimports.KindAppleHealthShortcut, UploadedVia: "connector", ObservedAt: &observedAt, CreatedAt: observedAt}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	receiptID := uuid.New()
	if err := db.Create(&models.SourceReceipt{ID: receiptID, SourceInstanceID: instanceID, RawFileID: rawID, SourceKind: coreimports.KindAppleHealthShortcut, IngestionMode: rawfiles.IngestionModeBoundedReplacement, ScopeJSON: []byte(`{}`), ObservedAt: &observedAt, ReceivedAt: observedAt, CreatedAt: observedAt}).Error; err != nil {
		t.Fatalf("create source receipt: %v", err)
	}
	importID, queueID := createImportReplay(t, db, rawID, coreimports.KindAppleHealthShortcut, observedAt)
	return rawID, importID, queueID
}

func createImportReplay(t *testing.T, db *gorm.DB, rawID uuid.UUID, parserKind string, createdAt time.Time) (uuid.UUID, uuid.UUID) {
	t.Helper()
	importID := uuid.New()
	queueID := uuid.New()
	if err := db.Create(&models.ImportJob{ID: importID, RawFileID: rawID, Status: StatusQueued, ParserKind: parserKind, ParserVersion: DefaultParserVersion, CreatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create import job: %v", err)
	}
	worker := "shortcut-integration"
	if err := db.Create(&models.Job{ID: queueID, Kind: jobs.KindAppleImportParse, Status: jobs.StatusRunning, PayloadJSON: []byte(`{}`), Attempts: 1, MaxAttempts: 3, RunAfter: createdAt, LockedBy: &worker, LockedAt: &createdAt, CreatedAt: createdAt, UpdatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create queue job: %v", err)
	}
	return importID, queueID
}

func processShortcutJob(t *testing.T, service *Service, db *gorm.DB, importID, queueID uuid.UUID) {
	t.Helper()
	ctx := jobs.WithClaim(context.Background(), jobs.Claim{JobID: queueID, WorkerID: "shortcut-integration", Attempt: 1})
	if err := service.ProcessContext(ctx, importID); err != nil {
		t.Fatalf("process shortcut import: %v", err)
	}
	var job models.ImportJob
	if err := db.First(&job, "id = ?", importID).Error; err != nil {
		t.Fatalf("load import job: %v", err)
	}
	if job.Status != StatusCompleted {
		t.Fatalf("import status = %q, want completed", job.Status)
	}
}
