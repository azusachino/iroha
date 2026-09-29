package parsers

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseHealthAutoExportFixture(t *testing.T) {
	path := filepath.Join("testdata", "health_auto_export.json")
	batch, err := ParseHealthAutoExport(path, "hae-test-hash", time.UTC)
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

// Coverage must carry the effective IANA timezone and span whole device-local
// days, whatever offset the device reported and whatever the server's TZ is.
func TestParseHealthAutoExportCoverageUsesEffectiveTimezone(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		date     string
		from, to time.Time
	}{
		// Same offset as the effective zone: one JST day.
		"tokyo": {"2026-09-20 00:00:00 +0900", time.Date(2026, 9, 20, 0, 0, 0, 0, tokyo), time.Date(2026, 9, 21, 0, 0, 0, 0, tokyo)},
		// China device day 20th runs 01:00 JST 20th to 01:00 JST 21st.
		"china": {"2026-09-20 00:00:00 +0800", time.Date(2026, 9, 20, 0, 0, 0, 0, tokyo), time.Date(2026, 9, 22, 0, 0, 0, 0, tokyo)},
		// US device day 20th runs 16:00 JST 20th to 16:00 JST 21st.
		"us-west": {"2026-09-20 00:00:00 -0700", time.Date(2026, 9, 20, 0, 0, 0, 0, tokyo), time.Date(2026, 9, 22, 0, 0, 0, 0, tokyo)},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "hae.json")
			body := `{"data":{"metrics":[{"name":"step_count","units":"count","data":[{"date":"` + tc.date + `","qty":1000}]}]}}`
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			batch, err := ParseHealthAutoExport(path, "hash", tokyo)
			if err != nil {
				t.Fatalf("ParseHealthAutoExport() failed: %v", err)
			}
			if len(batch.Coverage) != 1 {
				t.Fatalf("got %d coverage assertions, want 1", len(batch.Coverage))
			}
			cov := batch.Coverage[0]
			if cov.Timezone != "Asia/Tokyo" {
				t.Errorf("Timezone = %q, want Asia/Tokyo", cov.Timezone)
			}
			if !cov.From.Equal(tc.from) || !cov.To.Equal(tc.to) {
				t.Errorf("window = [%s, %s), want [%s, %s)", cov.From, cov.To, tc.from, tc.to)
			}
			if got := batch.Daily.Metrics[0].Day; !got.Equal(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)) {
				t.Errorf("metric Day = %s, want device-local 2026-09-20", got)
			}
		})
	}
}

func writeHAE(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "hae.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidateHealthAutoExportRejectsNonV2(t *testing.T) {
	for name, body := range map[string]string{
		"v1 workout without id":  `{"data":{"workouts":[{"name":"Running","start":"2026-09-20 07:00:00 +0900"}]}}`,
		"rfc3339 workout start":  `{"data":{"workouts":[{"id":"w1","name":"Running","start":"2026-09-20T07:00:00+09:00"}]}}`,
		"offsetless start":       `{"data":{"workouts":[{"id":"w1","name":"Running","start":"2026-09-20 07:00:00"}]}}`,
		"metric without name":    `{"data":{"metrics":[{"units":"count","data":[]}]}}`,
		"neither metrics nor wo": `{"data":{}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ValidateHealthAutoExport([]byte(body)); err == nil {
				t.Fatal("non-v2 payload accepted")
			}
		})
	}
}

func TestParseHealthAutoExportRejectsMalformedData(t *testing.T) {
	for name, body := range map[string]string{
		"metric date without offset": `{"data":{"metrics":[{"name":"step_count","units":"count","data":[{"date":"2026-09-20","qty":1}]}]}}`,
		"sleep without sleepEnd":     `{"data":{"metrics":[{"name":"sleep_analysis","units":"hr","data":[{"sleepStart":"2026-09-19 23:00:00 +0900","totalSleep":7}]}]}}`,
		"unsummarized sleep only":    `{"data":{"metrics":[{"name":"sleep_analysis","units":"hr","data":[{"startDate":"2026-09-19 23:00:00 +0900","endDate":"2026-09-20 00:00:00 +0900","value":"Core"}]}]}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC); err == nil {
				t.Fatal("malformed payload parsed without error")
			}
		})
	}
}

func TestParseHealthAutoExportConvertsUnitsForEveryPoint(t *testing.T) {
	body := `{"data":{"metrics":[{"name":"walking_running_distance","units":"mi","data":[
		{"date":"2026-09-20 00:00:00 +0900","qty":1},
		{"date":"2026-09-21 00:00:00 +0900","qty":2}]}],
	"workouts":[{"id":"w1","name":"Outdoor Run","start":"2026-09-20 07:00:00 +0900","end":"2026-09-20 07:30:00 +0900","duration":1800,
		"activeEnergyBurned":{"qty":418.4,"units":"kJ"}}]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC)
	if err != nil {
		t.Fatalf("ParseHealthAutoExport() failed: %v", err)
	}
	want := map[int]float64{20: 1.60934, 21: 3.21868}
	for _, m := range batch.Daily.Metrics {
		if m.Unit != "km" || math.Abs(m.Value-want[m.Day.Day()]) > 1e-9 {
			t.Errorf("day %d = %v %s, want %v km", m.Day.Day(), m.Value, m.Unit, want[m.Day.Day()])
		}
	}
	if kcal := batch.Activities[0].CaloriesKcal; kcal == nil || math.Abs(*kcal-100) > 1e-9 {
		t.Errorf("CaloriesKcal = %v, want 100", kcal)
	}
}

func TestParseHealthAutoExportReportsUnsupportedMetrics(t *testing.T) {
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	body := `{"data":{"metrics":[
{"name":"step_count","units":"count","data":[{"date":"2026-09-20 00:00:00 +0900","qty":1000}]},
{"name":"apple_stand_hour","units":"count","data":[{"date":"2026-09-20 00:00:00 +0900","qty":9},{"date":"2026-09-21 00:00:00 +0900","qty":8}]},
{"name":"time_in_daylight","units":"min","data":[{"date":"2026-09-20 00:00:00 +0900","qty":30}]}]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", tokyo)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Daily.Metrics) != 1 {
		t.Fatalf("got %d metrics, want only the supported step count", len(batch.Daily.Metrics))
	}
	cov := batch.Coverage[0]
	if cov.Completeness != "partial" {
		t.Errorf("Completeness = %q, want partial", cov.Completeness)
	}
	var scope struct {
		Unsupported map[string]int `json:"unsupported_metrics"`
	}
	if err := json.Unmarshal(cov.ScopeJSON, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.Unsupported["apple_stand_hour"] != 2 || scope.Unsupported["time_in_daylight"] != 1 || len(scope.Unsupported) != 2 {
		t.Errorf("unsupported = %v", scope.Unsupported)
	}
}

func TestParseHealthAutoExportReportsWhenEveryMetricIsUnsupported(t *testing.T) {
	body := `{"data":{"metrics":[{"name":"apple_stand_hour","units":"count","data":[{"date":"2026-09-20 00:00:00 +0900","qty":9}]}]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Coverage) != 1 || batch.Coverage[0].Completeness != "partial" {
		t.Fatalf("coverage = %+v", batch.Coverage)
	}
}
