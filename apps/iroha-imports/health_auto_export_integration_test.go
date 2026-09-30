//go:build integration

package imports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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

func TestIntegrationHealthAutoExportReplayAndBoundedReplacement(t *testing.T) {
	db := openImportsIntegrationDB(t)
	fixturePath := filepath.Join("..", "iroha-providers", "parsers", "testdata", "health_auto_export.json")
	body, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read hae fixture: %v", err)
	}

	partialBody := []byte(`{
  "data": {
    "metrics": [
      {
        "name": "step_count",
        "units": "count",
        "data": [
          {
            "date": "2026-09-21 09:00:00 +0900",
            "qty": 500,
            "source": "Apple Watch"
          }
        ]
      },
      {
        "name": "active_energy",
        "units": "kcal",
        "data": [{"date": "2026-09-21 00:00:00 +0900", "qty": 400}]
      },
      {
        "name": "apple_exercise_time",
        "units": "min",
        "data": [{"date": "2026-09-21 00:00:00 +0900", "qty": 30}]
      },
      {
        "name": "apple_stand_hour",
        "units": "count",
        "data": [{"date": "2026-09-21 00:00:00 +0900", "qty": 10}]
      },
      {
        "name": "time_in_daylight",
        "units": "min",
        "data": [{"date": "2026-09-21 00:00:00 +0900", "qty": 20}]
      }
    ],
    "workouts": []
  }
}`)

	instanceID := uuid.New()
	now := time.Now().UTC()
	sourceKey := "iphone-hae:primary:" + instanceID.String()

	queueIDs := make([]uuid.UUID, 0, 3)
	if err := db.Create(&models.SourceInstance{ID: instanceID, Provider: "health_auto_export", InstanceKey: sourceKey, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
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
	firstRawID, firstImportID, firstQueueID := createHaeEvidence(t, db, body, instanceID, now)
	cleanup = append(cleanup, firstRawID)
	queueIDs = append(queueIDs, firstQueueID)
	processHaeJob(t, service, db, firstImportID, firstQueueID)

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
	if coverageCount != 1 {
		t.Fatalf("coverage assertions after first import = %d, want 1", coverageCount)
	}
	var firstCoverage models.SourceCoverageAssertion
	if err := db.Where("source_receipt_id in (?)", db.Model(&models.SourceReceipt{}).Select("id").Where("raw_file_id = ?", firstRawID)).First(&firstCoverage).Error; err != nil {
		t.Fatalf("load first HAE coverage: %v", err)
	}
	if firstCoverage.Completeness != "unknown" {
		t.Fatalf("coverage completeness = %q, want unknown for a bounded supported payload", firstCoverage.Completeness)
	}
	var firstScope struct {
		Observed map[string]int `json:"observed_metrics"`
	}
	if err := json.Unmarshal(firstCoverage.ScopeJSON, &firstScope); err != nil {
		t.Fatalf("decode first coverage scope: %v", err)
	}
	if firstScope.Observed["step_count"] != 2 || firstScope.Observed["sleep_analysis"] != 3 {
		t.Errorf("observed metric inventory = %v", firstScope.Observed)
	}

	duplicateImportID, duplicateQueueID := createImportReplay(t, db, firstRawID, coreimports.KindHealthAutoExport, now.Add(time.Minute))
	queueIDs = append(queueIDs, duplicateQueueID)
	processHaeJob(t, service, db, duplicateImportID, duplicateQueueID)
	if err := db.Model(&models.SourceCoverageAssertion{}).Where("source_instance_id = ?", instanceID).Count(&coverageCount).Error; err != nil {
		t.Fatalf("count duplicate coverage: %v", err)
	}
	if coverageCount != 1 {
		t.Fatalf("coverage assertions after duplicate replay = %d, want 1", coverageCount)
	}

	partialRawID, partialImportID, partialQueueID := createHaeEvidence(t, db, partialBody, instanceID, now.Add(2*time.Minute))
	cleanup = append(cleanup, partialRawID)
	queueIDs = append(queueIDs, partialQueueID)
	processHaeJob(t, service, db, partialImportID, partialQueueID)
	if err := db.Model(&models.SourceObservation{}).Where("source_instance_id = ? and source_kind = ?", instanceID, "activity").Count(&observationCount).Error; err != nil {
		t.Fatalf("count partial observation: %v", err)
	}
	// Bounded replacement: older activity observation remains preserved!
	if observationCount != 1 {
		t.Fatalf("activity observations after partial import = %d, want 1", observationCount)
	}
	if err := db.Model(&models.SourceCoverageAssertion{}).Where("source_instance_id = ?", instanceID).Count(&coverageCount).Error; err != nil {
		t.Fatalf("count partial coverage: %v", err)
	}
	if coverageCount != 2 {
		t.Fatalf("coverage assertions after partial import = %d, want 2", coverageCount)
	}
	var importedSummary models.DailySummary
	if err := db.Where("first_raw_file_id = ?", partialRawID).First(&importedSummary).Error; err != nil {
		t.Fatalf("load imported HAE daily summary: %v", err)
	}
	if importedSummary.MoveKcal != 400 || importedSummary.ExerciseMin != 30 || importedSummary.StandHours != 10 {
		t.Errorf("imported summary = move %.0f, exercise %.0f, stand %.0f; want 400, 30, 10", importedSummary.MoveKcal, importedSummary.ExerciseMin, importedSummary.StandHours)
	}
	var partialCoverage models.SourceCoverageAssertion
	if err := db.Where("source_instance_id = ?", instanceID).Order("recorded_at desc").First(&partialCoverage).Error; err != nil {
		t.Fatalf("load partial HAE coverage: %v", err)
	}
	if partialCoverage.Completeness != "partial" {
		t.Fatalf("coverage completeness = %q, want partial", partialCoverage.Completeness)
	}
	var scope struct {
		Observed    map[string]int `json:"observed_metrics"`
		Unsupported map[string]int `json:"unsupported_metrics"`
	}
	if err := json.Unmarshal(partialCoverage.ScopeJSON, &scope); err != nil {
		t.Fatalf("decode coverage scope: %v", err)
	}
	if scope.Observed["active_energy"] != 1 || scope.Unsupported["time_in_daylight"] != 1 {
		t.Errorf("coverage scope observed=%v unsupported=%v", scope.Observed, scope.Unsupported)
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

func createHaeEvidence(t *testing.T, db *gorm.DB, body []byte, instanceID uuid.UUID, observedAt time.Time) (uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	rawID := uuid.New()
	hash := sha256.Sum256(body)
	storagePath := filepath.Join(t.TempDir(), "health-auto-export.json")
	if err := os.WriteFile(storagePath, body, 0o600); err != nil {
		t.Fatalf("write hae body: %v", err)
	}
	if err := db.Create(&models.RawFile{ID: rawID, SHA256: hex.EncodeToString(hash[:]), OriginalFilename: "health-auto-export.json", ContentType: "application/json", SizeBytes: int64(len(body)), StoragePath: storagePath, SourceKind: coreimports.KindHealthAutoExport, UploadedVia: "connector", ObservedAt: &observedAt, CreatedAt: observedAt}).Error; err != nil {
		t.Fatalf("create raw file: %v", err)
	}
	receiptID := uuid.New()
	if err := db.Create(&models.SourceReceipt{ID: receiptID, SourceInstanceID: instanceID, RawFileID: rawID, SourceKind: coreimports.KindHealthAutoExport, IngestionMode: rawfiles.IngestionModeBoundedReplacement, ScopeJSON: []byte(`{}`), ObservedAt: &observedAt, ReceivedAt: observedAt, CreatedAt: observedAt}).Error; err != nil {
		t.Fatalf("create source receipt: %v", err)
	}
	importID, queueID := createImportReplay(t, db, rawID, coreimports.KindHealthAutoExport, observedAt)
	return rawID, importID, queueID
}

func createImportReplay(t *testing.T, db *gorm.DB, rawID uuid.UUID, parserKind string, createdAt time.Time) (uuid.UUID, uuid.UUID) {
	t.Helper()
	importID := uuid.New()
	queueID := uuid.New()
	if err := db.Create(&models.ImportJob{ID: importID, RawFileID: rawID, Status: StatusQueued, ParserKind: parserKind, ParserVersion: DefaultParserVersion, CreatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create import job: %v", err)
	}
	worker := "hae-integration"
	if err := db.Create(&models.Job{ID: queueID, Kind: jobs.KindAppleImportParse, Status: jobs.StatusRunning, PayloadJSON: []byte(`{}`), Attempts: 1, MaxAttempts: 3, RunAfter: createdAt, LockedBy: &worker, LockedAt: &createdAt, CreatedAt: createdAt, UpdatedAt: createdAt}).Error; err != nil {
		t.Fatalf("create queue job: %v", err)
	}
	return importID, queueID
}

func processHaeJob(t *testing.T, service *Service, db *gorm.DB, importID, queueID uuid.UUID) {
	t.Helper()
	ctx := jobs.WithClaim(context.Background(), jobs.Claim{JobID: queueID, WorkerID: "hae-integration", Attempt: 1})
	if err := service.ProcessContext(ctx, importID); err != nil {
		t.Fatalf("process hae import: %v", err)
	}
	var job models.ImportJob
	if err := db.First(&job, "id = ?", importID).Error; err != nil {
		t.Fatalf("load import job: %v", err)
	}
	if job.Status != StatusCompleted {
		t.Fatalf("import status = %q, want completed", job.Status)
	}
}
