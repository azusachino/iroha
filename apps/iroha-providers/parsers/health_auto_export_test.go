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
	if act.SportType != "run" {
		t.Errorf("sportType = %q, want run", act.SportType)
	}
	if act.Title != "Outdoor Run" {
		t.Errorf("title = %q, want Outdoor Run", act.Title)
	}
	if act.DistanceM == nil || *act.DistanceM != 5120 {
		t.Errorf("distanceM = %v, want 5120", act.DistanceM)
	}
	if act.AvgPaceSPerKM == nil || math.Abs(*act.AvgPaceSPerKM-1872.0/5.12) > 0.001 {
		t.Errorf("avgPace = %v, want ~365.6", act.AvgPaceSPerKM)
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

func TestParseHaeWorkoutResolvesActivityTaxonomy(t *testing.T) {
	tests := []struct {
		name       string
		label      string
		hasRoute   bool
		wantSport  string
		wantTitle  string
		wantOrigin string
	}{
		{
			name:       "english outdoor run",
			label:      "Outdoor Run",
			hasRoute:   true,
			wantSport:  "run",
			wantTitle:  "Outdoor Run",
			wantOrigin: "system",
		},
		{
			name:       "japanese run with a route is outdoor",
			label:      "ラン",
			hasRoute:   true,
			wantSport:  "run",
			wantTitle:  "Outdoor Run",
			wantOrigin: "system",
		},
		{
			name:       "japanese run without route stays family level",
			label:      "ラン",
			wantSport:  "run",
			wantTitle:  "Run",
			wantOrigin: "system",
		},
		{
			name:       "japanese indoor run",
			label:      "屋内ラン",
			wantSport:  "run",
			wantTitle:  "Indoor Run",
			wantOrigin: "system",
		},
		{
			name:       "chinese outdoor run",
			label:      "户外跑步",
			wantSport:  "run",
			wantTitle:  "Outdoor Run",
			wantOrigin: "system",
		},
		{
			name:       "custom title is preserved verbatim",
			label:      "Sub-40 10k Tempo",
			wantSport:  "run",
			wantTitle:  "Sub-40 10k Tempo",
			wantOrigin: "custom",
		},
		{
			name:       "empty name falls back to the generic workout",
			label:      "",
			wantSport:  "other",
			wantTitle:  "Workout",
			wantOrigin: "system",
		},
		{
			name:       "unmapped sport keeps its own slug and title",
			label:      "Rowing",
			wantSport:  "rowing",
			wantTitle:  "Rowing",
			wantOrigin: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workout := haeWorkout{
				ID:       "workout-1",
				Name:     tt.label,
				Start:    "2026-09-20 07:00:00 +0900",
				End:      "2026-09-20 07:30:00 +0900",
				Duration: 1800,
			}
			if tt.hasRoute {
				workout.Route = []haeRoutePoint{{
					Latitude:  35.6812,
					Longitude: 139.7671,
					Timestamp: "2026-09-20 07:00:00 +0900",
				}}
			}

			activity, err := parseHaeWorkout(workout, "hae-hash")
			if err != nil {
				t.Fatalf("parseHaeWorkout() failed: %v", err)
			}
			if activity.SportType != tt.wantSport {
				t.Errorf("SportType = %q, want %q", activity.SportType, tt.wantSport)
			}
			if activity.Title != tt.wantTitle {
				t.Errorf("Title = %q, want %q", activity.Title, tt.wantTitle)
			}
			if activity.TitleOrigin != tt.wantOrigin {
				t.Errorf("TitleOrigin = %q, want %q", activity.TitleOrigin, tt.wantOrigin)
			}
			if got := len(activity.RoutePoints); tt.hasRoute && got != 1 {
				t.Errorf("got %d route points, want 1", got)
			}
			if activity.SourceKind != KindHealthAutoExport {
				t.Errorf("SourceKind = %q, want %q", activity.SourceKind, KindHealthAutoExport)
			}
		})
	}
}

func TestParseHealthAutoExportSleepEfficiencyUsesBedWindow(t *testing.T) {
	// inBedStart 23:30 → inBedEnd 07:01 is the honest bed window (7h31m);
	// the rounded inBed hours (7.5) match it here, but the session window
	// must widen to it and the ratio must use the window span.
	body := `{"data":{"metrics":[{"name":"sleep_analysis","units":"hr","data":[
		{"sleepStart":"2026-09-19 23:48:00 +0900","sleepEnd":"2026-09-20 06:52:00 +0900",
			"inBedStart":"2026-09-19 23:30:00 +0900","inBedEnd":"2026-09-20 07:01:00 +0900",
			"inBed":7.5,"asleep":6.4,"core":3.9,"deep":0.9,"rem":1.6,"awake":0.4}]}]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC)
	if err != nil {
		t.Fatalf("ParseHealthAutoExport() failed: %v", err)
	}
	if len(batch.Sleep) != 1 {
		t.Fatalf("got %d sleep sessions, want 1", len(batch.Sleep))
	}
	sleep := batch.Sleep[0]
	wantStart := time.Date(2026, 9, 19, 23, 30, 0, 0, tokyoOffset())
	if !sleep.StartedAt.Equal(wantStart) {
		t.Errorf("StartedAt = %s, want bed window open %s", sleep.StartedAt, wantStart)
	}
	wantEnd := time.Date(2026, 9, 20, 7, 1, 0, 0, tokyoOffset())
	if !sleep.EndedAt.Equal(wantEnd) {
		t.Errorf("EndedAt = %s, want bed window close %s", sleep.EndedAt, wantEnd)
	}
	if sleep.TimeInBedS != int((7*time.Hour+31*time.Minute)/time.Second) {
		t.Errorf("TimeInBedS = %d, want the 7h31m window", sleep.TimeInBedS)
	}
	// Asleep hours (6.4h exactly) over the exact window span (7h31m), not
	// the rounded inBed hours.
	wantEfficiency := (6.4 * 3600) / float64((7*time.Hour+31*time.Minute)/time.Second)
	if math.Abs(sleep.Efficiency-wantEfficiency) > 1e-9 {
		t.Errorf("Efficiency = %v, want %v", sleep.Efficiency, wantEfficiency)
	}
}

func TestParseHealthAutoExportSleepEfficiencyKeepsSummaryWithPartialSegments(t *testing.T) {
	// A stage timeline does not establish the complete in-bed interval.
	// Preserve the summary ratio even when the segment span differs.
	body := `{"data":{"metrics":[{"name":"sleep_analysis","units":"hr","data":[
		{"sleepStart":"2026-09-19 23:30:00 +0900","sleepEnd":"2026-09-20 07:00:00 +0900",
			"inBed":7,"asleep":7,"core":4,"deep":1,"rem":1.5,"awake":0.5},
		{"startDate":"2026-09-19 23:30:00 +0900","endDate":"2026-09-20 03:00:00 +0900","value":"Core"},
		{"startDate":"2026-09-20 03:00:00 +0900","endDate":"2026-09-20 03:30:00 +0900","value":"Awake"},
		{"startDate":"2026-09-20 03:30:00 +0900","endDate":"2026-09-20 07:00:00 +0900","value":"REM"}]}]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC)
	if err != nil {
		t.Fatalf("ParseHealthAutoExport() failed: %v", err)
	}
	if len(batch.Sleep) != 1 {
		t.Fatalf("got %d sleep sessions, want 1", len(batch.Sleep))
	}
	sleep := batch.Sleep[0]
	wantEfficiency := 1.0
	if math.Abs(sleep.Efficiency-wantEfficiency) > 1e-9 {
		t.Errorf("Efficiency = %v, want summary ratio %v", sleep.Efficiency, wantEfficiency)
	}
}

func tokyoOffset() *time.Location {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		panic(err)
	}
	return loc
}

func TestParseHealthAutoExportMapsMoveExerciseAndStandSummaries(t *testing.T) {
	body := `{"data":{"metrics":[
		{"name":"active_energy","units":"kJ","data":[
			{"date":"2026-09-20 00:00:00 +0900","qty":418.4},
			{"date":"2026-09-20 12:00:00 +0900","qty":418.4},
			{"date":"2026-09-21 00:00:00 +0900","qty":500}]},
		{"name":"apple_exercise_time","units":"min","data":[
			{"date":"2026-09-20 00:00:00 +0900","qty":30},
			{"date":"2026-09-20 12:00:00 +0900","qty":15}]},
		{"name":"apple_stand_hour","units":"count","data":[
			{"date":"2026-09-20 00:00:00 +0900","qty":9}]}
	]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC)
	if err != nil {
		t.Fatalf("ParseHealthAutoExport() failed: %v", err)
	}
	if len(batch.Daily.Summaries) != 1 {
		t.Fatalf("got %d daily summaries, want one merged day", len(batch.Daily.Summaries))
	}
	summary := batch.Daily.Summaries[0]
	if !summary.Day.Equal(time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Day = %s, want device-local 2026-09-20", summary.Day)
	}
	if math.Abs(summary.MoveKcal-200) > 1e-9 || summary.ExerciseMin != 45 || summary.StandHours != 9 {
		t.Errorf("summary = move %.2f kcal, exercise %.2f min, stand %.2f h; want 200, 45, 9", summary.MoveKcal, summary.ExerciseMin, summary.StandHours)
	}
	if summary.Source != KindHealthAutoExport {
		t.Errorf("Source = %q, want %q", summary.Source, KindHealthAutoExport)
	}
	// 2026-09-21 has only Active Energy, so its point is dropped and reported.
	cov := batch.Coverage[0]
	if cov.Completeness != "partial" {
		t.Errorf("Completeness = %q, want partial for the dropped incomplete day", cov.Completeness)
	}
	var scope struct {
		Unsupported map[string]int `json:"unsupported_metrics"`
	}
	if err := json.Unmarshal(cov.ScopeJSON, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.Unsupported["active_energy"] != 1 || len(scope.Unsupported) != 1 {
		t.Errorf("unsupported = %v, want the incomplete day's one active_energy point", scope.Unsupported)
	}
}

func TestParseHealthAutoExportMapsDocumentedBloodOxygenName(t *testing.T) {
	body := `{"data":{"metrics":[{"name":"blood_oxygen_saturation","units":"percent","data":[{"date":"2026-09-20 00:00:00 +0900","qty":98.5}]}]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC)
	if err != nil {
		t.Fatalf("ParseHealthAutoExport() failed: %v", err)
	}
	if len(batch.Daily.Metrics) != 1 {
		t.Fatalf("got %d daily metrics, want one blood oxygen metric", len(batch.Daily.Metrics))
	}
	metric := batch.Daily.Metrics[0]
	if metric.Metric != DailyMetricSpO2Avg || metric.Value != 98.5 || metric.Unit != "percent" {
		t.Errorf("metric = %+v, want %s 98.5 percent", metric, DailyMetricSpO2Avg)
	}
}

func TestParseHaeDailyMetricCanonicalizesVO2MaxUnit(t *testing.T) {
	for _, units := range []string{"ml/(kg·min)", "ml/(kg*min)", "ml/kg_min", "ml/kg·min", "ml/kg/min", ""} {
		metric := haeMetric{
			Name:  "vo2_max",
			Units: units,
			Data: []map[string]any{
				{"date": "2026-11-10 08:00:00 +0900", "qty": 38.9, "source": "Apple Watch"},
			},
		}
		results, _, _, err := parseHaeDailyMetric(metric)
		if err != nil {
			t.Fatalf("parseHaeDailyMetric(units=%q) failed: %v", units, err)
		}
		if len(results) != 1 {
			t.Fatalf("units=%q: got %d daily metrics, want 1", units, len(results))
		}
		if results[0].Unit != DailyMetricVO2MaxUnit {
			t.Errorf("units=%q: unit = %q, want %q", units, results[0].Unit, DailyMetricVO2MaxUnit)
		}
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
	if len(batch.Daily.Summaries) != 0 {
		t.Fatalf("got %d daily summaries, want none without all three ring metrics", len(batch.Daily.Summaries))
	}
	cov := batch.Coverage[0]
	if cov.Completeness != "partial" {
		t.Errorf("Completeness = %q, want partial", cov.Completeness)
	}
	var scope struct {
		Observed    map[string]int `json:"observed_metrics"`
		Unsupported map[string]int `json:"unsupported_metrics"`
	}
	if err := json.Unmarshal(cov.ScopeJSON, &scope); err != nil {
		t.Fatal(err)
	}
	if scope.Unsupported["apple_stand_hour"] != 2 || scope.Unsupported["time_in_daylight"] != 1 || len(scope.Unsupported) != 2 {
		t.Errorf("unsupported = %v", scope.Unsupported)
	}
	if scope.Observed["apple_stand_hour"] != 2 || scope.Observed["time_in_daylight"] != 1 || len(scope.Observed) != 3 {
		t.Errorf("observed = %v", scope.Observed)
	}
}

func TestParseHealthAutoExportReportsWhenEveryMetricIsUnsupported(t *testing.T) {
	body := `{"data":{"metrics":[{"name":"time_in_daylight","units":"min","data":[{"date":"2026-09-20 00:00:00 +0900","qty":30}]}]}}`
	batch, err := ParseHealthAutoExport(writeHAE(t, body), "hash", time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	if len(batch.Coverage) != 1 || batch.Coverage[0].Completeness != "partial" {
		t.Fatalf("coverage = %+v", batch.Coverage)
	}
}
