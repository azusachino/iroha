package parsers

import (
	"path/filepath"
	"testing"
	"time"
)

func TestParseHealthAutoExportFixture(t *testing.T) {
	path := filepath.Join("testdata", "health_auto_export.json")
	batch, err := ParseHealthAutoExport(path, "hae-test-hash")
	if err != nil {
		t.Fatalf("ParseHealthAutoExport() failed: %v", err)
	}

	// 1. Verify Daily Metrics
	if len(batch.Daily.Metrics) != 2 {
		t.Fatalf("got %d daily metrics, want 2", len(batch.Daily.Metrics))
	}
	var stepsMetric, hrMetric *DailyMetricObservation
	for i := range batch.Daily.Metrics {
		m := &batch.Daily.Metrics[i]
		if m.Metric == DailyMetricSteps {
			stepsMetric = m
		}
		if m.Metric == DailyMetricRestingHR {
			hrMetric = m
		}
	}
	if stepsMetric == nil {
		t.Fatal("missing steps metric")
	}
	// 112 + 131 = 243
	if stepsMetric.Value != 243 {
		t.Errorf("steps value = %v, want 243", stepsMetric.Value)
	}
	if hrMetric == nil {
		t.Fatal("missing resting HR metric")
	}
	if hrMetric.Value != 64 {
		t.Errorf("resting HR value = %v, want 64", hrMetric.Value)
	}

	// 2. Verify Sleep
	if len(batch.Sleep) != 1 {
		t.Fatalf("got %d sleep sessions, want 1", len(batch.Sleep))
	}
	sleep := batch.Sleep[0]
	expectedWakeDate := time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)
	if !sleep.WakeDate.Equal(expectedWakeDate) {
		t.Errorf("wakeDate = %v, want %v", sleep.WakeDate, expectedWakeDate)
	}
	if sleep.AsleepS != int(6.4*3600) {
		t.Errorf("asleepS = %d, want %d", sleep.AsleepS, int(6.4*3600))
	}
	if sleep.CoreS != int(3.9*3600) {
		t.Errorf("coreS = %d, want %d", sleep.CoreS, int(3.9*3600))
	}
	if len(sleep.Segments) != 2 {
		t.Errorf("got %d sleep segments, want 2", len(sleep.Segments))
	}

	// 3. Verify Activities (Workouts)
	if len(batch.Activities) != 1 {
		t.Fatalf("got %d activities, want 1", len(batch.Activities))
	}
	act := batch.Activities[0]
	if act.ExternalID != "7C1D3E0A-4B2F-4E8A-9D11-2A6B5C3F9E01" {
		t.Errorf("externalID = %q, want 7C1D3E0A-4B2F-4E8A-9D11-2A6B5C3F9E01", act.ExternalID)
	}
	if act.SportType != "running" {
		t.Errorf("sportType = %q, want running", act.SportType)
	}
	if act.DistanceM == nil || *act.DistanceM != 5120 {
		t.Errorf("distanceM = %v, want 5120", act.DistanceM)
	}
	if act.DurationS == nil || *act.DurationS != 1872 {
		t.Errorf("durationS = %v, want 1872", act.DurationS)
	}
	if act.AvgHR == nil || *act.AvgHR != 145 {
		t.Errorf("avgHR = %v, want 145", act.AvgHR)
	}
	if act.CaloriesKcal == nil || *act.CaloriesKcal != 402.3 {
		t.Errorf("caloriesKcal = %v, want 402.3", act.CaloriesKcal)
	}
	if len(act.RoutePoints) != 2 {
		t.Errorf("got %d route points, want 2", len(act.RoutePoints))
	}
	if len(act.Samplings) != 2 {
		t.Errorf("got %d samplings, want 2", len(act.Samplings))
	}

	// 4. Verify Coverage
	if len(batch.Coverage) != 1 {
		t.Fatalf("got %d coverage assertions, want 1", len(batch.Coverage))
	}
	cov := batch.Coverage[0]
	if cov.Category != "health" || cov.IngestionMode != "bounded_replacement" {
		t.Errorf("coverage mismatch: %+v", cov)
	}
}

func TestValidateHealthAutoExport(t *testing.T) {
	valid := []byte(`{"data":{"metrics":[{"name":"step_count","data":[]}]}}`)
	meta, err := ValidateHealthAutoExport(valid)
	if err != nil {
		t.Fatalf("ValidateHealthAutoExport() failed: %v", err)
	}
	if meta.SourceInstanceKey != DefaultHealthAutoExportInstance {
		t.Errorf("SourceInstanceKey = %q, want %q", meta.SourceInstanceKey, DefaultHealthAutoExportInstance)
	}

	invalid := []byte(`{"invalid": true}`)
	if _, err := ValidateHealthAutoExport(invalid); err == nil {
		t.Error("expected error for missing data object, got nil")
	}
}
