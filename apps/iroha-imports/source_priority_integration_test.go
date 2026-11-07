//go:build integration

package imports

import (
	"errors"
	"testing"
	"time"

	"github.com/azusachino/iroha/apps/iroha-core/observations"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestIntegrationDailyMetricNativeThenHAE(t *testing.T) {
	db := openImportsIntegrationDB(t)
	errRollbackFixture := errors.New("rollback fixture")
	err := db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		type evidence struct {
			raw      models.RawFile
			snapshot uuid.UUID
		}
		create := func(kind string) evidence {
			t.Helper()
			raw := models.RawFile{ID: uuid.New(), SHA256: uuid.NewString(), OriginalFilename: "fixture", StoragePath: "/tmp/fixture", SourceKind: kind, UploadedVia: "integration", CreatedAt: now}
			instance := models.SourceInstance{ID: uuid.New(), Provider: kind, InstanceKey: uuid.NewString(), CreatedAt: now, UpdatedAt: now}
			receipt := models.SourceReceipt{ID: uuid.New(), SourceInstanceID: instance.ID, RawFileID: raw.ID, SourceKind: kind, IngestionMode: "full_snapshot", ScopeJSON: []byte(`{}`), ReceivedAt: now, CreatedAt: now}
			job := models.ImportJob{ID: uuid.New(), RawFileID: raw.ID, Status: StatusCompleted, ParserKind: kind, ParserVersion: DefaultParserVersion, CreatedAt: now}
			snapshot := models.ImportSnapshot{ID: uuid.New(), ImportJobID: job.ID, RawFileID: raw.ID, SHA256: uuid.NewString(), ParserVersion: DefaultParserVersion, CreatedAt: now}
			for _, row := range []any{&raw, &instance, &receipt, &job, &snapshot} {
				if err := tx.Create(row).Error; err != nil {
					t.Fatalf("create fixture: %v", err)
				}
			}
			return evidence{raw: raw, snapshot: snapshot.ID}
		}
		native, hae := create("apple_health_export"), create("health_auto_export")
		metric := observations.DailyMetric{Day: time.Date(2040, 2, 1, 0, 0, 0, 0, time.UTC), Metric: "steps", Value: 100, Unit: "count", Source: "native"}
		service := &Service{}
		if err := service.persistDailyMetric(tx, native.raw, metric, native.snapshot, false); err != nil {
			t.Fatal(err)
		}
		metric.Value, metric.Source = 200, "hae"
		if err := service.persistDailyMetric(tx, hae.raw, metric, hae.snapshot, false); err != nil {
			t.Fatal(err)
		}
		var got models.DailyMetric
		if err := tx.Where("day = ? and metric = ?", metric.Day, metric.Metric).First(&got).Error; err != nil {
			t.Fatal(err)
		}
		if got.Value != 100 || got.Source != "native" {
			t.Fatalf("selected value = %v (%s), want native 100", got.Value, got.Source)
		}
		var count int64
		if err := tx.Model(&models.DailyMetricObservation{}).Where("daily_metric_id = ?", got.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("observations = %d, want 2", count)
		}
		metric.Day = metric.Day.AddDate(0, 0, 1)
		if err := service.persistDailyMetric(tx, hae.raw, metric, hae.snapshot, false); err != nil {
			t.Fatal(err)
		}
		metric.Value, metric.Source = 100, "native"
		if err := service.persistDailyMetric(tx, native.raw, metric, native.snapshot, false); err != nil {
			t.Fatal(err)
		}
		var reverse models.DailyMetric
		if err := tx.Where("day = ? and metric = ?", metric.Day, metric.Metric).First(&reverse).Error; err != nil {
			t.Fatal(err)
		}
		if reverse.Value != 100 || reverse.Source != "native" {
			t.Fatalf("reverse selected value = %v (%s), want native 100", reverse.Value, reverse.Source)
		}
		// A native zero is a real value and must survive a later HAE import.
		metric.Day = metric.Day.AddDate(0, 0, 1)
		metric.Value, metric.Source = 0, "native"
		if err := service.persistDailyMetric(tx, native.raw, metric, native.snapshot, false); err != nil {
			t.Fatal(err)
		}
		metric.Value, metric.Source = 200, "hae"
		if err := service.persistDailyMetric(tx, hae.raw, metric, hae.snapshot, false); err != nil {
			t.Fatal(err)
		}
		var zero models.DailyMetric
		if err := tx.Where("day = ? and metric = ?", metric.Day, metric.Metric).First(&zero).Error; err != nil {
			t.Fatal(err)
		}
		if zero.Value != 0 || zero.Source != "native" {
			t.Fatalf("selected zero = %v (%s), want native zero", zero.Value, zero.Source)
		}
		metric.Day = metric.Day.AddDate(0, 0, 1)
		metric.Value, metric.Source, metric.Unit = 100, "native", "count"
		if err := service.persistDailyMetric(tx, native.raw, metric, native.snapshot, false); err != nil {
			t.Fatal(err)
		}
		metric.Value, metric.Source, metric.Unit = 200, "hae", "different-unit"
		if err := service.persistDailyMetric(tx, hae.raw, metric, hae.snapshot, false); err != nil {
			t.Fatal(err)
		}
		var incompatible models.DailyMetric
		if err := tx.Where("day = ? and metric = ?", metric.Day, metric.Metric).First(&incompatible).Error; err != nil {
			t.Fatal(err)
		}
		if incompatible.Value != 100 || incompatible.Unit != "count" || incompatible.Source != "native" {
			t.Fatalf("incompatible selected value = %v %s (%s), want native 100 count", incompatible.Value, incompatible.Unit, incompatible.Source)
		}
		if err := service.persistDailyMetric(tx, hae.raw, metric, hae.snapshot, true); err != nil {
			t.Fatal(err)
		}
		var replayed models.DailyMetric
		if err := tx.Where("id = ?", incompatible.ID).First(&replayed).Error; err != nil {
			t.Fatal(err)
		}
		if replayed.Value != 100 || replayed.Unit != "count" || replayed.Source != "native" {
			t.Fatalf("replayed selected value = %v %s (%s), want native 100 count", replayed.Value, replayed.Unit, replayed.Source)
		}
		var replayCount int64
		if err := tx.Model(&models.DailyMetricObservation{}).Where("daily_metric_id = ?", incompatible.ID).Count(&replayCount).Error; err != nil {
			t.Fatal(err)
		}
		if replayCount != 2 {
			t.Fatalf("observations after replay = %d, want 2", replayCount)
		}
		return errRollbackFixture
	})
	if err != errRollbackFixture {
		t.Fatalf("fixture transaction: %v", err)
	}
}
