//go:build integration

package imports

import (
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-providers/parsers"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestIntegrationReplayPreservesSelectedAndOtherSource(t *testing.T) {
	db := openImportsIntegrationDB(t)
	now := time.Now().UTC()
	metric := parsers.DailyMetricObservation{
		Day:    time.Date(2026, time.January, 2, 0, 0, 0, 0, time.UTC),
		Metric: parsers.DailyMetricRestingHR,
		Value:  57.5,
		Unit:   "count/min",
		Source: "Watch A",
	}
	sourceKey := dailyMetricSourceKey(metric)
	metricB := metric
	metricB.Value = 61
	metricB.Source = "Watch B"

	type fixture struct {
		rawFileID, instanceID, receiptID, jobID, snapshotID uuid.UUID
	}
	newFixture := func(name string) fixture {
		t.Helper()
		f := fixture{rawFileID: uuid.New(), instanceID: uuid.New(), receiptID: uuid.New(), jobID: uuid.New(), snapshotID: uuid.New()}
		if err := db.Create(&models.RawFile{
			ID: f.rawFileID, SHA256: name + "-" + f.rawFileID.String(), OriginalFilename: name + ".zip",
			StoragePath: "/tmp/" + name + ".zip", SourceKind: parsers.KindAppleHealthExport,
			UploadedVia: "integration", CreatedAt: now,
		}).Error; err != nil {
			t.Fatalf("create %s raw file: %v", name, err)
		}
		if err := db.Create(&models.SourceInstance{ID: f.instanceID, Provider: parsers.KindAppleHealthExport, InstanceKey: name + "-" + f.instanceID.String(), CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatalf("create %s source instance: %v", name, err)
		}
		if err := db.Create(&models.SourceReceipt{ID: f.receiptID, SourceInstanceID: f.instanceID, RawFileID: f.rawFileID, SourceKind: parsers.KindAppleHealthExport, IngestionMode: "full_snapshot", ScopeJSON: []byte(`{}`), ReceivedAt: now, CreatedAt: now}).Error; err != nil {
			t.Fatalf("create %s source receipt: %v", name, err)
		}
		if err := db.Create(&models.ImportJob{ID: f.jobID, RawFileID: f.rawFileID, Status: StatusCompleted, ParserKind: parsers.KindAppleHealthExport, ParserVersion: DefaultParserVersion, CreatedAt: now}).Error; err != nil {
			t.Fatalf("create %s import job: %v", name, err)
		}
		if err := db.Create(&models.ImportSnapshot{ID: f.snapshotID, ImportJobID: f.jobID, RawFileID: f.rawFileID, SHA256: name + "-snapshot", ParserVersion: DefaultParserVersion, CreatedAt: now}).Error; err != nil {
			t.Fatalf("create %s import snapshot: %v", name, err)
		}
		return f
	}
	a := newFixture("replay-a")
	b := newFixture("replay-b")
	t.Cleanup(func() {
		_ = db.Exec("delete from tb_daily_metrics where day = ? and metric = ?", metric.Day, metric.Metric).Error
		_ = db.Exec("delete from tb_source_observations where raw_file_id in ?", []uuid.UUID{a.rawFileID, b.rawFileID}).Error
		_ = db.Exec("delete from tb_import_snapshots where raw_file_id in ?", []uuid.UUID{a.rawFileID, b.rawFileID}).Error
		_ = db.Exec("delete from tb_import_jobs where id in ?", []uuid.UUID{a.jobID, b.jobID}).Error
		_ = db.Exec("delete from tb_source_receipts where id in ?", []uuid.UUID{a.receiptID, b.receiptID}).Error
		_ = db.Exec("delete from tb_source_instances where id in ?", []uuid.UUID{a.instanceID, b.instanceID}).Error
		_ = db.Exec("delete from tb_raw_files where id in ?", []uuid.UUID{a.rawFileID, b.rawFileID}).Error
	})

	service := &Service{}
	persist := func(rawFileID, snapshotID uuid.UUID, value parsers.DailyMetricObservation, reprocess bool) {
		t.Helper()
		if err := db.Transaction(func(tx *gorm.DB) error {
			return service.persistDailyMetric(tx, models.RawFile{ID: rawFileID}, value, snapshotID, reprocess)
		}); err != nil {
			t.Fatalf("persist metric: %v", err)
		}
	}

	persist(a.rawFileID, a.snapshotID, metric, false)
	var canonical models.DailyMetric
	if err := db.Where("day = ? and metric = ?", metric.Day, metric.Metric).First(&canonical).Error; err != nil {
		t.Fatalf("load first canonical metric: %v", err)
	}
	canonicalID := canonical.ID

	var observationA models.SourceObservation
	if err := db.Where("source_instance_id = ? and source_kind = ? and source_key = ?", a.instanceID, "daily_metric", sourceKey).First(&observationA).Error; err != nil {
		t.Fatalf("load source A observation: %v", err)
	}

	persist(b.rawFileID, b.snapshotID, metricB, false)
	var observationB models.SourceObservation
	if err := db.Where("source_instance_id = ? and source_kind = ? and source_key = ?", b.instanceID, "daily_metric", sourceKey).First(&observationB).Error; err != nil {
		t.Fatalf("load source B observation: %v", err)
	}
	if observationA.ID == observationB.ID {
		t.Fatal("source-scoped observations collapsed across source instances")
	}
	var afterB models.DailyMetric
	if err := db.First(&afterB, "id = ?", canonicalID).Error; err != nil {
		t.Fatalf("load B-selected metric: %v", err)
	}
	if afterB.SelectedObservationID == nil || *afterB.SelectedObservationID != observationB.ID {
		t.Fatalf("selected observation after B = %v, want %v", afterB.SelectedObservationID, observationB.ID)
	}

	replaySnapshot := models.ImportSnapshot{ID: uuid.New(), ImportJobID: a.jobID, RawFileID: a.rawFileID, SHA256: "replay-a-again", ParserVersion: "reprocess", CreatedAt: now.Add(time.Second)}
	if err := db.Create(&replaySnapshot).Error; err != nil {
		t.Fatalf("create replay snapshot: %v", err)
	}
	persist(a.rawFileID, replaySnapshot.ID, metric, true)

	var afterReplay models.DailyMetric
	if err := db.First(&afterReplay, "id = ?", canonicalID).Error; err != nil {
		t.Fatalf("load replayed metric: %v", err)
	}
	if afterReplay.ID != canonicalID {
		t.Fatalf("canonical ID after replay = %v, want %v", afterReplay.ID, canonicalID)
	}
	if afterReplay.SelectedObservationID == nil || *afterReplay.SelectedObservationID != observationB.ID {
		t.Fatalf("selected observation after A replay = %v, want B %v", afterReplay.SelectedObservationID, observationB.ID)
	}
	if afterReplay.Value != metricB.Value || afterReplay.Source != metricB.Source {
		t.Fatalf("canonical metric after A replay = %v/%q, want B %v/%q", afterReplay.Value, afterReplay.Source, metricB.Value, metricB.Source)
	}
	var observationCount int64
	if err := db.Model(&models.SourceObservation{}).Where("source_kind = ? and source_key = ? and source_instance_id in ?", "daily_metric", sourceKey, []uuid.UUID{a.instanceID, b.instanceID}).Count(&observationCount).Error; err != nil {
		t.Fatalf("count source observations: %v", err)
	}
	if observationCount != 2 {
		t.Fatalf("source observations after replay = %d, want 2", observationCount)
	}
}
