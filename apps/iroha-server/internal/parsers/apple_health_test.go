package parsers

import (
	"archive/zip"
	"os"
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
