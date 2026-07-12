package parsers

import (
	"archive/zip"
	"os"
	"strings"
	"testing"
	"time"
)

const sampleAppleExport = `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:00:00 -0700" value="120"/>
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalDistance="5" totalDistanceUnit="km" sourceName="Watch" sourceVersion="10.0" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700">
    <MetadataEntry key="HKIndoorWorkout" value="0"/>
    <WorkoutEvent type="HKWorkoutEventTypePause" date="2024-01-01 08:10:00 -0700" duration="1" durationUnit="min"/>
    <WorkoutStatistics type="HKQuantityTypeIdentifierDistanceWalkingRunning" sum="5" average="0.5" maximum="1" minimum="0" unit="km" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700"/>
    <WorkoutStatistics type="HKQuantityTypeIdentifierActiveEnergyBurned" sum="300" unit="kcal" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700"/>
    <WorkoutRoute>
      <FileReference path="/workout-routes/route_2024-01-01_8.00am.gpx"/>
    </WorkoutRoute>
  </Workout>
</HealthData>
`

func TestDecodeAppleWorkoutsCapturesNestedSubtree(t *testing.T) {
	decoder := newAppleWorkoutDecoderForTest(t, sampleAppleExport)

	if len(decoder.workouts) != 1 {
		t.Fatalf("expected 1 workout, got %d", len(decoder.workouts))
	}
	workout := decoder.workouts[0]

	if workout.SourceName != "Watch" || workout.SourceVersion != "10.0" {
		t.Fatalf("expected workout-level source attrs to be captured, got %+v", workout)
	}

	if workout.WorkoutRoute == nil || workout.WorkoutRoute.FileReference == nil {
		t.Fatalf("expected WorkoutRoute/FileReference to be populated")
	}
	if got, want := workout.WorkoutRoute.FileReference.Path, "/workout-routes/route_2024-01-01_8.00am.gpx"; got != want {
		t.Errorf("FileReference path = %q, want %q", got, want)
	}

	if len(workout.Statistics) != 2 {
		t.Fatalf("expected 2 WorkoutStatistics entries, got %d", len(workout.Statistics))
	}
	if workout.Statistics[0].Type != "HKQuantityTypeIdentifierDistanceWalkingRunning" || workout.Statistics[0].Sum != "5" {
		t.Errorf("unexpected first statistic: %+v", workout.Statistics[0])
	}
	if workout.Statistics[1].Type != "HKQuantityTypeIdentifierActiveEnergyBurned" || workout.Statistics[1].Sum != "300" {
		t.Errorf("unexpected second statistic: %+v", workout.Statistics[1])
	}

	if len(workout.Events) != 1 {
		t.Fatalf("expected 1 WorkoutEvent, got %d", len(workout.Events))
	}
	if workout.Events[0].Type != "HKWorkoutEventTypePause" || workout.Events[0].Duration != "1" {
		t.Errorf("unexpected event: %+v", workout.Events[0])
	}

	if len(workout.MetadataEntries) != 1 {
		t.Fatalf("expected 1 MetadataEntry, got %d", len(workout.MetadataEntries))
	}
	if workout.MetadataEntries[0].Key != "HKIndoorWorkout" || workout.MetadataEntries[0].Value != "0" {
		t.Errorf("unexpected metadata entry: %+v", workout.MetadataEntries[0])
	}
}

func TestDecodeAppleSleepSegmentsStreamsSelectedStages(t *testing.T) {
	const sleepXML = `<HealthData>
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" startDate="2024-01-01 22:00:00 -0700" endDate="2024-01-01 22:01:00 -0700" value="60"/>
  <Record type="HKCategoryTypeIdentifierSleepAnalysis" value="HKCategoryValueSleepAnalysisInBed" sourceName="Watch" startDate="2024-01-01 22:00:00 -0700" endDate="2024-01-02 06:00:00 -0700"/>
  <Record type="HKCategoryTypeIdentifierSleepAnalysis" value="HKCategoryValueSleepAnalysisAsleepCore" sourceName="Watch" startDate="2024-01-01 23:00:00 -0700" endDate="2024-01-02 01:00:00 -0700"/>
  <Record type="HKCategoryTypeIdentifierSleepAnalysis" value="HKCategoryValueSleepAnalysisAsleepDeep" sourceName="Watch" startDate="2024-01-02 00:30:00 -0700" endDate="2024-01-02 02:00:00 -0700"/>
</HealthData>`

	segments, err := decodeAppleSleepSegments(strings.NewReader(sleepXML))
	if err != nil {
		t.Fatalf("decodeAppleSleepSegments returned error: %v", err)
	}
	if len(segments) != 3 {
		t.Fatalf("decoded %d sleep segments, want 3", len(segments))
	}
	if segments[0].Stage != SleepStageInBed || segments[1].Stage != SleepStageCore || segments[2].Stage != SleepStageDeep {
		t.Fatalf("unexpected stages: %+v", segments)
	}
	if segments[0].Source != "Watch" {
		t.Errorf("source = %q, want Watch", segments[0].Source)
	}
}

func TestDecodeAppleDailyActivityRollsUpRingsAndPriorityIntervals(t *testing.T) {
	const dailyXML = `<HealthData>
  <ActivitySummary dateComponents="2024-01-01" activeEnergyBurned="600" activeEnergyBurnedGoal="500" appleExerciseTime="45" appleExerciseTimeGoal="30" appleStandHours="10" appleStandHoursGoal="12"/>
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="iPhone" unit="count" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 09:00:00 -0700" value="200"/>
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="Watch" unit="count" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 09:00:00 -0700" value="100"/>
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="iPhone" unit="count" startDate="2024-01-01 09:00:00 -0700" endDate="2024-01-01 10:00:00 -0700" value="50"/>
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="Third Party" unit="count" startDate="2024-01-01 08:30:00 -0700" endDate="2024-01-01 09:30:00 -0700" value="300"/>
  <Record type="HKQuantityTypeIdentifierDistanceWalkingRunning" sourceName="Watch" unit="mi" startDate="2024-01-01 10:00:00 -0700" endDate="2024-01-01 10:30:00 -0700" value="1"/>
  <Record type="HKQuantityTypeIdentifierFlightsClimbed" sourceName="Watch" unit="count" startDate="2024-01-01 11:00:00 -0700" endDate="2024-01-01 11:01:00 -0700" value="3"/>
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 11:00:00 -0700" endDate="2024-01-01 11:01:00 -0700" value="60"/>
</HealthData>`

	summaries, records, err := decodeAppleDailyActivity(strings.NewReader(dailyXML))
	if err != nil {
		t.Fatalf("decodeAppleDailyActivity returned error: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("decoded %d summaries, want 1", len(summaries))
	}
	if summaries[0].MoveKcal != 600 || summaries[0].ExerciseMin != 45 || summaries[0].StandGoalHours != 12 {
		t.Errorf("summary = %+v, want direct ActivitySummary values", summaries[0])
	}
	if summaries[0].Source != KindAppleHealthExport {
		t.Errorf("summary source = %q, want %q", summaries[0].Source, KindAppleHealthExport)
	}

	metrics := rollupDailyMetrics(records)
	if len(metrics) != 3 {
		t.Fatalf("rolled up %d metrics, want 3: %+v", len(metrics), metrics)
	}
	byMetric := make(map[string]ParsedDailyMetric, len(metrics))
	for _, metric := range metrics {
		byMetric[metric.Metric] = metric
	}
	steps := byMetric[DailyMetricSteps]
	if steps.Metric != DailyMetricSteps || steps.Value != 150 || steps.Source != "Watch" {
		t.Errorf("steps metric = %+v, want watch 100 + non-overlapping iPhone 50", steps)
	}
	distance := byMetric[DailyMetricDistanceKM]
	if distance.Metric != DailyMetricDistanceKM || distance.Unit != "km" || distance.Value != 1.609344 {
		t.Errorf("distance metric = %+v, want converted miles", distance)
	}
	flights := byMetric[DailyMetricFlights]
	if flights.Value != 3 {
		t.Errorf("flights metric = %+v, want 3", flights)
	}
}

func TestRollupDailyMetricsKeepsHybridIntervals(t *testing.T) {
	day := "2024-01-01"
	start := sleepTestTime("2024-01-01 08:00:00 -0700")
	metrics := rollupDailyMetrics([]dailyActivityRecord{
		{day: day, metric: DailyMetricSteps, unit: "count", value: 100, startedAt: start, endedAt: start.Add(time.Hour), source: "Watch"},
		{day: day, metric: DailyMetricSteps, unit: "count", value: 200, startedAt: start.Add(30 * time.Minute), endedAt: start.Add(90 * time.Minute), source: "iPhone"},
		{day: day, metric: DailyMetricSteps, unit: "count", value: 50, startedAt: start.Add(time.Hour), endedAt: start.Add(2 * time.Hour), source: "iPhone"},
	})
	if len(metrics) != 1 || metrics[0].Value != 150 {
		t.Fatalf("hybrid rollup = %+v, want 150 from watch + non-overlapping iPhone interval", metrics)
	}
}

func TestRollupDailyMetricsReducesVitals(t *testing.T) {
	const dailyXML = `<HealthData>
  <Record type="HKQuantityTypeIdentifierRestingHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:00:00 -0700" value="55"/>
  <Record type="HKQuantityTypeIdentifierRestingHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 20:00:00 -0700" endDate="2024-01-01 20:00:00 -0700" value="60"/>
  <Record type="HKQuantityTypeIdentifierHeartRateVariabilitySDNN" sourceName="Watch" unit="ms" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:00:00 -0700" value="40"/>
  <Record type="HKQuantityTypeIdentifierHeartRateVariabilitySDNN" sourceName="Watch" unit="ms" startDate="2024-01-01 20:00:00 -0700" endDate="2024-01-01 20:00:00 -0700" value="60"/>
  <Record type="HKQuantityTypeIdentifierOxygenSaturation" sourceName="Watch" unit="%" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:00:00 -0700" value="0.95"/>
  <Record type="HKQuantityTypeIdentifierOxygenSaturation" sourceName="Watch" unit="%" startDate="2024-01-01 20:00:00 -0700" endDate="2024-01-01 20:00:00 -0700" value="0.98"/>
  <Record type="HKQuantityTypeIdentifierBodyFatPercentage" sourceName="Watch" unit="%" startDate="2024-01-01 20:00:00 -0700" endDate="2024-01-01 20:00:00 -0700" value="20"/>
</HealthData>`

	_, records, err := decodeAppleDailyActivity(strings.NewReader(dailyXML))
	if err != nil {
		t.Fatalf("decodeAppleDailyActivity returned error: %v", err)
	}
	metrics := rollupDailyMetrics(records)
	byMetric := make(map[string]ParsedDailyMetric, len(metrics))
	for _, metric := range metrics {
		byMetric[metric.Metric] = metric
	}
	if len(metrics) != 4 {
		t.Fatalf("rolled up %d metrics, want 4: %+v", len(metrics), metrics)
	}
	if got := byMetric[DailyMetricRestingHR]; got.Value != 60 || got.Unit != "count/min" {
		t.Errorf("resting HR = %+v, want latest value 60 count/min", got)
	}
	if got := byMetric[DailyMetricHRVSDNN]; got.Value != 50 || got.Unit != "ms" {
		t.Errorf("HRV = %+v, want average 50 ms", got)
	}
	if got := byMetric[DailyMetricSpO2Avg]; got.Value != 96.5 || got.Unit != "%" {
		t.Errorf("SpO2 average = %+v, want average normalized value 96.5%%", got)
	}
	if got := byMetric[DailyMetricSpO2Min]; got.Value != 95 || got.Unit != "%" {
		t.Errorf("SpO2 minimum = %+v, want 95%%", got)
	}
}

func TestBuildSleepSessionUsesOverlapSafeRollups(t *testing.T) {
	segments := []ParsedSleepSegment{
		{Stage: SleepStageInBed, StartedAt: sleepTestTime("2024-01-01 22:00:00 -0700"), EndedAt: sleepTestTime("2024-01-02 04:00:00 -0700"), Source: "Watch"},
		{Stage: SleepStageCore, StartedAt: sleepTestTime("2024-01-01 23:00:00 -0700"), EndedAt: sleepTestTime("2024-01-02 01:00:00 -0700"), Source: "Watch"},
		{Stage: SleepStageDeep, StartedAt: sleepTestTime("2024-01-02 00:30:00 -0700"), EndedAt: sleepTestTime("2024-01-02 02:00:00 -0700"), Source: "Watch"},
		{Stage: SleepStageREM, StartedAt: sleepTestTime("2024-01-02 02:00:00 -0700"), EndedAt: sleepTestTime("2024-01-02 03:00:00 -0700"), Source: "Watch"},
	}

	session := buildSleepSession(segments)
	if session.TimeInBedS != 21600 {
		t.Errorf("time in bed = %d, want 21600", session.TimeInBedS)
	}
	if session.AsleepS != 14400 {
		t.Errorf("asleep = %d, want 14400 after unioning overlapping stages", session.AsleepS)
	}
	if session.CoreS != 7200 || session.DeepS != 5400 || session.RemS != 3600 {
		t.Errorf("stage rollups = core %d, deep %d, rem %d; want 7200, 5400, 3600", session.CoreS, session.DeepS, session.RemS)
	}
	wakeYear, wakeMonth, wakeDay := session.WakeDate.Date()
	endYear, endMonth, endDay := session.EndedAt.Date()
	if !session.IsMainSleep || wakeYear != endYear || wakeMonth != endMonth || wakeDay != endDay {
		t.Errorf("main sleep = %v, wake date = %v, ended at = %v", session.IsMainSleep, session.WakeDate, session.EndedAt)
	}
	if session.Efficiency != 2.0/3.0 {
		t.Errorf("efficiency = %v, want %v", session.Efficiency, 2.0/3.0)
	}
}

func TestSessionizeSleepSegmentsSplitsAfterConfiguredGap(t *testing.T) {
	segments := []ParsedSleepSegment{
		{Stage: SleepStageAsleepUnspecified, StartedAt: sleepTestTime("2024-01-01 22:00:00 -0700"), EndedAt: sleepTestTime("2024-01-01 23:00:00 -0700")},
		{Stage: SleepStageAsleepUnspecified, StartedAt: sleepTestTime("2024-01-02 00:01:00 -0700"), EndedAt: sleepTestTime("2024-01-02 01:00:00 -0700")},
	}

	sessions := sessionizeSleepSegments(segments, time.Hour)
	if len(sessions) != 2 {
		t.Fatalf("sessionized into %d sessions, want 2", len(sessions))
	}
}

func sleepTestTime(value string) time.Time {
	parsed, err := time.Parse("2006-01-02 15:04:05 -0700", value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestDecodeAppleWorkoutsProducesUnchangedActivity(t *testing.T) {
	activities, err := decodeAppleWorkouts(strings.NewReader(sampleAppleExport))
	if err != nil {
		t.Fatalf("decodeAppleWorkouts returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity (stray top-level Record must be skipped), got %d", len(activities))
	}

	activity := activities[0]
	if activity.Provider != "apple_health" {
		t.Errorf("Provider = %q, want apple_health", activity.Provider)
	}
	wantExternalID := "Watch||HKWorkoutActivityTypeRunning|2024-01-01 08:00:00 -0700|2024-01-01 08:30:00 -0700|30"
	if activity.ExternalID != wantExternalID {
		t.Errorf("ExternalID = %q, want %q", activity.ExternalID, wantExternalID)
	}
	if activity.SourceActivityID != wantExternalID {
		t.Errorf("SourceActivityID = %q, want %q", activity.SourceActivityID, wantExternalID)
	}
	if activity.ContentHash == "" {
		t.Errorf("ContentHash should not be empty for an apple_health workout")
	}
	if activity.SportType != "run" {
		t.Errorf("SportType = %q, want run", activity.SportType)
	}
	if activity.SourceKind != "apple_health_export" {
		t.Errorf("SourceKind = %q, want apple_health_export", activity.SourceKind)
	}
	if activity.DurationS == nil || *activity.DurationS != 1800 {
		t.Errorf("DurationS = %v, want 1800", activity.DurationS)
	}
	if activity.DistanceM == nil || *activity.DistanceM != 5000 {
		t.Errorf("DistanceM = %v, want 5000", activity.DistanceM)
	}
	if activity.CaloriesKcal == nil || *activity.CaloriesKcal != 300 {
		t.Errorf("CaloriesKcal = %v, want 300", activity.CaloriesKcal)
	}
	if activity.EndedAt == nil {
		t.Errorf("EndedAt should not be nil")
	}
	if len(activity.RoutePoints) != 0 {
		t.Errorf("RoutePoints should be empty in this task (route linking is task-4), got %d", len(activity.RoutePoints))
	}
}

func TestStableDeviceKeyStripsVolatilePointerAndCreationDate(t *testing.T) {
	// Two "exports" of the same physical Apple Watch: the runtime pointer
	// after "HKDevice:" and the "creation date:" both differ, as they do
	// between real re-exports, but name/hardware/manufacturer are the same.
	export1 := "<<HKDevice: 0x8dcef18c0>, name:Apple Watch, manufacturer:Apple Inc., model:Watch, hardware:Watch7,1, software:10.2, creation date:2024-01-01 08:30:00 -0700>>"
	export2 := "<<HKDevice: 0x7f2a1b400>, name:Apple Watch, manufacturer:Apple Inc., model:Watch, hardware:Watch7,1, software:10.2, creation date:2024-06-15 09:00:00 -0700>>"

	key1 := stableDeviceKey(export1)
	key2 := stableDeviceKey(export2)

	if key1 == "" {
		t.Fatalf("stableDeviceKey returned empty string for a populated device attr")
	}
	if key1 != key2 {
		t.Errorf("stableDeviceKey differed across exports of the same device: %q != %q", key1, key2)
	}
	if strings.Contains(key1, "0x") {
		t.Errorf("stableDeviceKey leaked the volatile pointer token: %q", key1)
	}
	if strings.Contains(key1, "creation date") {
		t.Errorf("stableDeviceKey leaked the volatile creation date: %q", key1)
	}

	// A genuinely different device (different hardware) must produce a
	// different key.
	otherHardware := "<<HKDevice: 0x8dcef18c0>, name:Apple Watch, manufacturer:Apple Inc., model:Watch, hardware:Watch6,1, software:9.0, creation date:2024-01-01 08:30:00 -0700>>"
	if stableDeviceKey(otherHardware) == key1 {
		t.Errorf("stableDeviceKey should differ for a different hardware model")
	}

	if got := stableDeviceKey(""); got != "" {
		t.Errorf("stableDeviceKey(\"\") = %q, want empty string", got)
	}
}

func TestWorkoutStableKeyAndContentHashAcrossReexports(t *testing.T) {
	// Same workout appearing in two different full exports: the device
	// string's pointer/creation-date differ (as with a real re-export), but
	// everything else about the workout is identical.
	deviceA := "<<HKDevice: 0x8dcef18c0>, name:Apple Watch, manufacturer:Apple Inc., hardware:Watch7,1, software:10.2, creation date:2024-01-01 08:30:00 -0700>>"
	deviceB := "<<HKDevice: 0x1122aabb0>, name:Apple Watch, manufacturer:Apple Inc., hardware:Watch7,1, software:10.2, creation date:2024-06-01 12:00:00 -0700>>"

	base := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		TotalDistance:       "5",
		TotalDistanceUnit:   "km",
		SourceName:          "Watch",
		SourceVersion:       "10.0",
		Statistics: []appleWorkoutStatistic{
			{Type: "HKQuantityTypeIdentifierDistanceWalkingRunning", Sum: "5", Unit: "km"},
		},
		MetadataEntries: []appleMetadataEntry{
			{Key: "HKIndoorWorkout", Value: "0"},
			// Real exports duplicate metadata keys under one Workout; the
			// hash must dedupe by key rather than treating this as extra
			// content.
			{Key: "HKIndoorWorkout", Value: "0"},
		},
	}

	export1 := base
	export1.Device = deviceA
	export2 := base
	export2.Device = deviceB

	key1 := stableWorkoutSourceKey(export1)
	key2 := stableWorkoutSourceKey(export2)
	if key1 != key2 {
		t.Errorf("stableWorkoutSourceKey differed across re-exports of the same workout: %q != %q", key1, key2)
	}

	hash1 := workoutContentHash(export1)
	hash2 := workoutContentHash(export2)
	if hash1 != hash2 {
		t.Errorf("workoutContentHash differed across re-exports of the same workout: %q != %q", hash1, hash2)
	}
	if hash1 == "" {
		t.Errorf("workoutContentHash should not be empty")
	}

	// Now change something that actually affects the workout content (a
	// statistic value) and confirm the hash changes but the identity key
	// does not - a content update to an already-known workout, not a new
	// workout.
	changed := export2
	changed.Statistics = []appleWorkoutStatistic{
		{Type: "HKQuantityTypeIdentifierDistanceWalkingRunning", Sum: "6", Unit: "km"},
	}
	changedKey := stableWorkoutSourceKey(changed)
	changedHash := workoutContentHash(changed)

	if changedKey != key1 {
		t.Errorf("stableWorkoutSourceKey should be unaffected by a statistics change, got %q want %q", changedKey, key1)
	}
	if changedHash == hash1 {
		t.Errorf("workoutContentHash should change when workout statistics change")
	}
}

const sampleRouteGPX = `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1"><trk><trkseg>
<trkpt lat="37.1" lon="-122.1"><ele>10.0</ele><time>2024-01-01T15:00:00Z</time></trkpt>
<trkpt lat="37.2" lon="-122.2"><ele>12.0</ele><time>2024-01-01T15:01:00Z</time></trkpt>
</trkseg></trk></gpx>
`

// buildAppleHealthZip writes an in-memory zip with the given export.xml body
// and optional extra file (name -> contents) to a temp file, returning its
// path.
func buildAppleHealthZip(t *testing.T, exportXML string, extraFiles map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	zipPath := dir + "/export.zip"
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create temp zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	writeEntry := func(name, contents string) {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %q: %v", name, err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatalf("failed to write zip entry %q: %v", name, err)
		}
	}

	writeEntry("apple_health_export/export.xml", exportXML)
	for name, contents := range extraFiles {
		writeEntry(name, contents)
	}

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return zipPath
}

func TestParseAppleHealthExportAttachesRouteToWorkout(t *testing.T) {
	exportXML := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalDistance="5" totalDistanceUnit="km" sourceName="Watch" sourceVersion="10.0" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700">
    <WorkoutRoute>
      <FileReference path="/workout-routes/route_x.gpx"/>
      <MetadataEntry key="HKMetadataKeySyncIdentifier" value="ABC"/>
    </WorkoutRoute>
  </Workout>
</HealthData>
`
	zipPath := buildAppleHealthZip(t, exportXML, map[string]string{
		"apple_health_export/workout-routes/route_x.gpx": sampleRouteGPX,
	})

	activities, err := ParseAppleHealthExport(zipPath, "rawhash123")
	if err != nil {
		t.Fatalf("ParseAppleHealthExport returned error: %v", err)
	}

	if len(activities) != 1 {
		t.Fatalf("expected exactly 1 activity (route must NOT become a standalone gpx activity), got %d: %+v", len(activities), activities)
	}

	activity := activities[0]
	if activity.Provider != "apple_health" {
		t.Errorf("Provider = %q, want apple_health", activity.Provider)
	}
	if len(activity.RoutePoints) != 2 {
		t.Fatalf("expected 2 route points attached, got %d", len(activity.RoutePoints))
	}
	if activity.RoutePoints[0].Lat != 37.1 || activity.RoutePoints[0].Lon != -122.1 {
		t.Errorf("unexpected first route point: %+v", activity.RoutePoints[0])
	}
	if activity.RoutePoints[1].Lat != 37.2 || activity.RoutePoints[1].Lon != -122.2 {
		t.Errorf("unexpected second route point: %+v", activity.RoutePoints[1])
	}
}

func TestParseAppleHealthExportWorkoutWithoutRoute(t *testing.T) {
	exportXML := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalDistance="5" totalDistanceUnit="km" sourceName="Watch" sourceVersion="10.0" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700">
  </Workout>
</HealthData>
`
	zipPath := buildAppleHealthZip(t, exportXML, nil)

	activities, err := ParseAppleHealthExport(zipPath, "rawhash123")
	if err != nil {
		t.Fatalf("ParseAppleHealthExport returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if len(activities[0].RoutePoints) != 0 {
		t.Errorf("expected empty RoutePoints for workout without a route, got %d", len(activities[0].RoutePoints))
	}
}

func TestParseAppleHealthExportMissingRouteFileIsSkippedGracefully(t *testing.T) {
	exportXML := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalDistance="5" totalDistanceUnit="km" sourceName="Watch" sourceVersion="10.0" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700">
    <WorkoutRoute>
      <FileReference path="/workout-routes/route_missing.gpx"/>
    </WorkoutRoute>
  </Workout>
</HealthData>
`
	zipPath := buildAppleHealthZip(t, exportXML, nil)

	activities, err := ParseAppleHealthExport(zipPath, "rawhash123")
	if err != nil {
		t.Fatalf("ParseAppleHealthExport returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected import to proceed despite missing route file, got %d activities", len(activities))
	}
	if len(activities[0].RoutePoints) != 0 {
		t.Errorf("expected empty RoutePoints when referenced route file is missing, got %d", len(activities[0].RoutePoints))
	}
}

func TestParseGPXPoints(t *testing.T) {
	points, err := parseGPXPoints(strings.NewReader(sampleRouteGPX))
	if err != nil {
		t.Fatalf("parseGPXPoints returned error: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(points))
	}
	if points[0].Lat != 37.1 || points[0].Lon != -122.1 {
		t.Errorf("unexpected first point: %+v", points[0])
	}
	if points[0].ElevationM == nil || *points[0].ElevationM != 10.0 {
		t.Errorf("unexpected elevation on first point: %+v", points[0].ElevationM)
	}
	if points[0].Ts == nil {
		t.Errorf("expected first point to have a parsed timestamp")
	}
}

func TestWorkoutContentHashChangesWithRouteIdentity(t *testing.T) {
	base := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		SourceName:          "Watch",
	}

	noRoute := base
	hashNoRoute := workoutContentHash(noRoute)

	withRoute := base
	withRoute.WorkoutRoute = &appleWorkoutRoute{
		FileReference: &appleFileReference{Path: "/workout-routes/route_x.gpx"},
		MetadataEntries: []appleMetadataEntry{
			{Key: "HKMetadataKeySyncIdentifier", Value: "ABC"},
		},
	}
	hashWithRoute := workoutContentHash(withRoute)

	if hashWithRoute == hashNoRoute {
		t.Errorf("workoutContentHash should change when a route is added")
	}

	changedPath := base
	changedPath.WorkoutRoute = &appleWorkoutRoute{
		FileReference: &appleFileReference{Path: "/workout-routes/route_y.gpx"},
		MetadataEntries: []appleMetadataEntry{
			{Key: "HKMetadataKeySyncIdentifier", Value: "ABC"},
		},
	}
	hashChangedPath := workoutContentHash(changedPath)
	if hashChangedPath == hashWithRoute {
		t.Errorf("workoutContentHash should change when route path changes")
	}

	changedSyncID := base
	changedSyncID.WorkoutRoute = &appleWorkoutRoute{
		FileReference: &appleFileReference{Path: "/workout-routes/route_x.gpx"},
		MetadataEntries: []appleMetadataEntry{
			{Key: "HKMetadataKeySyncIdentifier", Value: "XYZ"},
		},
	}
	hashChangedSyncID := workoutContentHash(changedSyncID)
	if hashChangedSyncID == hashWithRoute {
		t.Errorf("workoutContentHash should change when route sync identifier changes")
	}
}

func TestWorkoutToActivityAppliesStatisticsToSummaryFields(t *testing.T) {
	workout := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		TotalDistance:       "1",
		TotalDistanceUnit:   "km",
		SourceName:          "Watch",
		Statistics: []appleWorkoutStatistic{
			{Type: "HKQuantityTypeIdentifierHeartRate", Average: "145.4", Maximum: "172.6", Minimum: "98", Unit: "count/min"},
			{Type: "HKQuantityTypeIdentifierDistanceWalkingRunning", Sum: "5", Unit: "km"},
			{Type: "HKQuantityTypeIdentifierActiveEnergyBurned", Sum: "300", Unit: "kcal"},
		},
	}

	activity, ok := workoutToActivity(workout)
	if !ok {
		t.Fatalf("workoutToActivity returned ok=false")
	}

	if activity.AvgHR == nil || *activity.AvgHR != 145 {
		t.Errorf("AvgHR = %v, want 145 (rounded from 145.4)", activity.AvgHR)
	}
	if activity.MaxHR == nil || *activity.MaxHR != 173 {
		t.Errorf("MaxHR = %v, want 173 (rounded from 172.6)", activity.MaxHR)
	}
	// The per-statistic distance (5km) must win over the workout-level
	// totalDistance attribute (1km).
	if activity.DistanceM == nil || *activity.DistanceM != 5000 {
		t.Errorf("DistanceM = %v, want 5000 (from statistic, not totalDistance)", activity.DistanceM)
	}
	// duration 1800s over 5km -> 360 s/km.
	if activity.AvgPaceSPerKM == nil || *activity.AvgPaceSPerKM != 360 {
		t.Errorf("AvgPaceSPerKM = %v, want 360", activity.AvgPaceSPerKM)
	}
}

func TestWorkoutToActivityWithoutDistanceStatKeepsTotalDistance(t *testing.T) {
	workout := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		TotalDistance:       "1",
		TotalDistanceUnit:   "km",
		SourceName:          "Watch",
	}

	activity, ok := workoutToActivity(workout)
	if !ok {
		t.Fatalf("workoutToActivity returned ok=false")
	}
	if activity.DistanceM == nil || *activity.DistanceM != 1000 {
		t.Errorf("DistanceM = %v, want 1000 (from workout-level totalDistance)", activity.DistanceM)
	}
	if activity.AvgHR != nil || activity.MaxHR != nil {
		t.Errorf("expected AvgHR/MaxHR to remain nil without a HeartRate statistic, got %v/%v", activity.AvgHR, activity.MaxHR)
	}
}

func TestDeriveWorkoutLapsFromLapEvents(t *testing.T) {
	workout := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		SourceName:          "Watch",
		Events: []appleWorkoutEvent{
			{Type: "HKWorkoutEventTypeLap", Date: "2024-01-01 08:10:00 -0700"},
			{Type: "HKWorkoutEventTypeLap", Date: "2024-01-01 08:22:00 -0700"},
		},
	}

	activity, ok := workoutToActivity(workout)
	if !ok {
		t.Fatalf("workoutToActivity returned ok=false")
	}

	if len(activity.Laps) != 2 {
		t.Fatalf("expected 2 laps, got %d: %+v", len(activity.Laps), activity.Laps)
	}

	lap1 := activity.Laps[0]
	if lap1.LapNo != 1 {
		t.Errorf("lap1.LapNo = %d, want 1", lap1.LapNo)
	}
	if lap1.StartTs == nil || lap1.StartTs.Format("15:04:05") != "08:00:00" {
		t.Errorf("lap1.StartTs = %v, want workout start (08:00:00)", lap1.StartTs)
	}
	if lap1.EndTs == nil || lap1.EndTs.Format("15:04:05") != "08:10:00" {
		t.Errorf("lap1.EndTs = %v, want first lap event (08:10:00)", lap1.EndTs)
	}
	if lap1.DurationS == nil || *lap1.DurationS != 600 {
		t.Errorf("lap1.DurationS = %v, want 600", lap1.DurationS)
	}

	lap2 := activity.Laps[1]
	if lap2.LapNo != 2 {
		t.Errorf("lap2.LapNo = %d, want 2", lap2.LapNo)
	}
	if lap2.StartTs == nil || lap2.StartTs.Format("15:04:05") != "08:10:00" {
		t.Errorf("lap2.StartTs = %v, want prior lap boundary (08:10:00)", lap2.StartTs)
	}
	if lap2.EndTs == nil || lap2.EndTs.Format("15:04:05") != "08:22:00" {
		t.Errorf("lap2.EndTs = %v, want second lap event (08:22:00)", lap2.EndTs)
	}
	if lap2.DurationS == nil || *lap2.DurationS != 720 {
		t.Errorf("lap2.DurationS = %v, want 720", lap2.DurationS)
	}
}

func TestDeriveWorkoutLapsMarkerOnlyProducesNoLaps(t *testing.T) {
	workout := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		SourceName:          "Watch",
		Events: []appleWorkoutEvent{
			{Type: "HKWorkoutEventTypeMarker", Date: "2024-01-01 08:15:00 -0700"},
		},
	}

	activity, ok := workoutToActivity(workout)
	if !ok {
		t.Fatalf("workoutToActivity returned ok=false")
	}
	if activity.Laps != nil {
		t.Errorf("expected nil Laps for a marker-only workout, got %+v", activity.Laps)
	}
}

func TestDeriveWorkoutLapsFallsBackToSegmentEvents(t *testing.T) {
	workout := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		SourceName:          "Watch",
		Events: []appleWorkoutEvent{
			{Type: "HKWorkoutEventTypeSegment", Date: "2024-01-01 08:15:00 -0700"},
		},
	}

	activity, ok := workoutToActivity(workout)
	if !ok {
		t.Fatalf("workoutToActivity returned ok=false")
	}
	if len(activity.Laps) != 1 {
		t.Fatalf("expected 1 lap derived from the Segment event, got %d", len(activity.Laps))
	}
}

func TestWorkoutContentHashChangesWithEvents(t *testing.T) {
	base := appleWorkout{
		WorkoutActivityType: "HKWorkoutActivityTypeRunning",
		StartDate:           "2024-01-01 08:00:00 -0700",
		EndDate:             "2024-01-01 08:30:00 -0700",
		Duration:            "30",
		DurationUnit:        "min",
		SourceName:          "Watch",
	}
	baseHash := workoutContentHash(base)

	withEvent := base
	withEvent.Events = []appleWorkoutEvent{
		{Type: "HKWorkoutEventTypeLap", Date: "2024-01-01 08:10:00 -0700"},
	}
	withEventHash := workoutContentHash(withEvent)
	if withEventHash == baseHash {
		t.Errorf("workoutContentHash should change when an event is added")
	}

	changedEventDate := base
	changedEventDate.Events = []appleWorkoutEvent{
		{Type: "HKWorkoutEventTypeLap", Date: "2024-01-01 08:11:00 -0700"},
	}
	changedEventDateHash := workoutContentHash(changedEventDate)
	if changedEventDateHash == withEventHash {
		t.Errorf("workoutContentHash should change when an event's date changes")
	}
}

func TestParseAppleHealthExportAssociatesSelectedRecordsToWorkoutWindow(t *testing.T) {
	// Document order mirrors a real export: Records appear before the
	// Workout element, so pass 2 must re-stream to find owning windows only
	// after all workouts (and hence windows) are known.
	exportXML := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 08:05:00 -0700" endDate="2024-01-01 08:05:00 -0700" value="120"/>
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 08:15:00 -0700" endDate="2024-01-01 08:15:00 -0700" value="135"/>
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 09:00:00 -0700" endDate="2024-01-01 09:00:00 -0700" value="70"/>
  <Record type="HKQuantityTypeIdentifierStepCount" sourceName="Watch" unit="count" startDate="2024-01-01 08:06:00 -0700" endDate="2024-01-01 08:06:00 -0700" value="50"/>
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalDistance="5" totalDistanceUnit="km" sourceName="Watch" sourceVersion="10.0" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700">
  </Workout>
</HealthData>
`
	zipPath := buildAppleHealthZip(t, exportXML, nil)

	activities, err := ParseAppleHealthExport(zipPath, "rawhash123")
	if err != nil {
		t.Fatalf("ParseAppleHealthExport returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}

	samplings := activities[0].Samplings
	if len(samplings) != 2 {
		t.Fatalf("expected 2 in-window heart_rate samplings (outside-window HR and non-selected StepCount must be excluded), got %d: %+v", len(samplings), samplings)
	}

	for _, s := range samplings {
		if s.SamplingType != "heart_rate" {
			t.Errorf("SamplingType = %q, want heart_rate", s.SamplingType)
		}
		if s.Unit != "count/min" {
			t.Errorf("Unit = %q, want count/min", s.Unit)
		}
	}
	if samplings[0].Value != 120 || samplings[1].Value != 135 {
		t.Errorf("unexpected sampling values: %+v", samplings)
	}

	wantTs0, _ := parseAppleTime("2024-01-01 08:05:00 -0700")
	wantTs1, _ := parseAppleTime("2024-01-01 08:15:00 -0700")
	if !samplings[0].Ts.Equal(wantTs0) {
		t.Errorf("samplings[0].Ts = %v, want %v", samplings[0].Ts, wantTs0)
	}
	if !samplings[1].Ts.Equal(wantTs1) {
		t.Errorf("samplings[1].Ts = %v, want %v", samplings[1].Ts, wantTs1)
	}
}

func TestParseAppleHealthExportIncludesRecordsExactlyOnWindowBoundary(t *testing.T) {
	exportXML := `<?xml version="1.0" encoding="UTF-8"?>
<HealthData locale="en_US">
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:00:00 -0700" value="100"/>
  <Record type="HKQuantityTypeIdentifierHeartRate" sourceName="Watch" unit="count/min" startDate="2024-01-01 08:30:00 -0700" endDate="2024-01-01 08:30:00 -0700" value="140"/>
  <Workout workoutActivityType="HKWorkoutActivityTypeRunning" duration="30" durationUnit="min" totalDistance="5" totalDistanceUnit="km" sourceName="Watch" sourceVersion="10.0" startDate="2024-01-01 08:00:00 -0700" endDate="2024-01-01 08:30:00 -0700">
  </Workout>
</HealthData>
`
	zipPath := buildAppleHealthZip(t, exportXML, nil)

	activities, err := ParseAppleHealthExport(zipPath, "rawhash123")
	if err != nil {
		t.Fatalf("ParseAppleHealthExport returned error: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}

	samplings := activities[0].Samplings
	if len(samplings) != 2 {
		t.Fatalf("expected both boundary records (exactly at start and exactly at end) to be included, got %d: %+v", len(samplings), samplings)
	}
}

func TestFindOwningActivityBinarySearchEdges(t *testing.T) {
	mkTime := func(s string) time.Time {
		ts, err := parseAppleTime(s)
		if err != nil {
			t.Fatalf("parseAppleTime(%q): %v", s, err)
		}
		return ts
	}

	act1 := &ParsedActivity{Title: "first"}
	act2 := &ParsedActivity{Title: "second"}
	windows := []workoutWindow{
		{start: mkTime("2024-01-01 08:00:00 -0700"), end: mkTime("2024-01-01 08:30:00 -0700"), activity: act1},
		{start: mkTime("2024-01-01 09:00:00 -0700"), end: mkTime("2024-01-01 09:15:00 -0700"), activity: act2},
	}

	cases := []struct {
		name string
		ts   string
		want *ParsedActivity
	}{
		{"before all windows", "2024-01-01 07:00:00 -0700", nil},
		{"exactly first window start", "2024-01-01 08:00:00 -0700", act1},
		{"inside first window", "2024-01-01 08:15:00 -0700", act1},
		{"exactly first window end", "2024-01-01 08:30:00 -0700", act1},
		{"gap between windows", "2024-01-01 08:45:00 -0700", nil},
		{"exactly second window start", "2024-01-01 09:00:00 -0700", act2},
		{"inside second window", "2024-01-01 09:10:00 -0700", act2},
		{"exactly second window end", "2024-01-01 09:15:00 -0700", act2},
		{"after all windows", "2024-01-01 10:00:00 -0700", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findOwningActivity(windows, mkTime(tc.ts))
			if got != tc.want {
				t.Errorf("findOwningActivity(%s) = %v, want %v", tc.ts, got, tc.want)
			}
		})
	}
}

// newAppleWorkoutDecoderForTest is a tiny helper that exposes the decoded
// appleWorkout structs (not just the resulting ParsedActivity) so the test
// can assert on the newly captured nested fields.
type testAppleWorkoutCapture struct {
	workouts []appleWorkout
}

func newAppleWorkoutDecoderForTest(t *testing.T, xmlBody string) testAppleWorkoutCapture {
	t.Helper()
	workouts, err := decodeAppleWorkoutsRaw(strings.NewReader(xmlBody))
	if err != nil {
		t.Fatalf("decodeAppleWorkoutsRaw returned error: %v", err)
	}
	return testAppleWorkoutCapture{workouts: workouts}
}
