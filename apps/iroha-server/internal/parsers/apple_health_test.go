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
	activities, err := decodeAppleWorkouts(strings.NewReader(sampleAppleExport), "rawhash123")
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
	wantExternalID := "rawhash123:HKWorkoutActivityTypeRunning:2024-01-01 08:00:00 -0700"
	if activity.ExternalID != wantExternalID {
		t.Errorf("ExternalID = %q, want %q", activity.ExternalID, wantExternalID)
	}
	if activity.SourceActivityID != wantExternalID {
		t.Errorf("SourceActivityID = %q, want %q", activity.SourceActivityID, wantExternalID)
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
