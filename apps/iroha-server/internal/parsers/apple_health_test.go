package parsers

import (
	"strings"
	"testing"
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
