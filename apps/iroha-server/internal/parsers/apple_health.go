package parsers

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type appleWorkout struct {
	WorkoutActivityType string `xml:"workoutActivityType,attr"`
	StartDate           string `xml:"startDate,attr"`
	EndDate             string `xml:"endDate,attr"`
	Duration            string `xml:"duration,attr"`
	DurationUnit        string `xml:"durationUnit,attr"`
	TotalDistance       string `xml:"totalDistance,attr"`
	TotalDistanceUnit   string `xml:"totalDistanceUnit,attr"`
	SourceName          string `xml:"sourceName,attr"`
	SourceVersion       string `xml:"sourceVersion,attr"`
	Device              string `xml:"device,attr"`

	WorkoutRoute    *appleWorkoutRoute      `xml:"WorkoutRoute"`
	Statistics      []appleWorkoutStatistic `xml:"WorkoutStatistics"`
	Events          []appleWorkoutEvent     `xml:"WorkoutEvent"`
	MetadataEntries []appleMetadataEntry    `xml:"MetadataEntry"`
}

type appleWorkoutRoute struct {
	FileReference   *appleFileReference  `xml:"FileReference"`
	MetadataEntries []appleMetadataEntry `xml:"MetadataEntry"`
}

// syncIdentifier returns the WorkoutRoute's HKMetadataKeySyncIdentifier
// metadata value, if present. This is a stable identity for the route
// distinct from the workout's own identity, used to detect route
// additions/changes/removals via the workout content hash.
func (route *appleWorkoutRoute) syncIdentifier() string {
	if route == nil {
		return ""
	}
	for _, entry := range route.MetadataEntries {
		if entry.Key == "HKMetadataKeySyncIdentifier" {
			return entry.Value
		}
	}
	return ""
}

type appleFileReference struct {
	Path string `xml:"path,attr"`
}

type appleWorkoutStatistic struct {
	Type      string `xml:"type,attr"`
	Sum       string `xml:"sum,attr"`
	Average   string `xml:"average,attr"`
	Maximum   string `xml:"maximum,attr"`
	Minimum   string `xml:"minimum,attr"`
	Unit      string `xml:"unit,attr"`
	StartDate string `xml:"startDate,attr"`
	EndDate   string `xml:"endDate,attr"`
}

type appleWorkoutEvent struct {
	Type         string `xml:"type,attr"`
	Date         string `xml:"date,attr"`
	Duration     string `xml:"duration,attr"`
	DurationUnit string `xml:"durationUnit,attr"`
}

type appleMetadataEntry struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

func ParseAppleHealthExport(path string, rawHash string) ([]ParsedActivity, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var activities []ParsedActivity
	for _, file := range reader.File {
		if !strings.HasSuffix(file.Name, "export.xml") {
			continue
		}

		workouts, err := parseAppleWorkoutsRaw(file)
		if err != nil {
			return nil, err
		}

		for _, workout := range workouts {
			activity, ok := workoutToActivity(workout)
			if !ok {
				continue
			}
			attachWorkoutRoute(&activity, workout, reader)
			activities = append(activities, activity)
		}
	}
	return activities, nil
}

func parseAppleWorkoutsRaw(file *zip.File) ([]appleWorkout, error) {
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	return decodeAppleWorkoutsRaw(opened)
}

// attachWorkoutRoute locates the zip entry referenced by the workout's
// WorkoutRoute/FileReference (if any), parses its track points, and attaches
// them to activity.RoutePoints. Workouts without a route, or whose
// referenced gpx entry is missing from the zip, are left with empty
// RoutePoints - this must never fail the overall import.
func attachWorkoutRoute(activity *ParsedActivity, workout appleWorkout, reader *zip.ReadCloser) {
	if workout.WorkoutRoute == nil || workout.WorkoutRoute.FileReference == nil {
		return
	}
	refPath := workout.WorkoutRoute.FileReference.Path
	if refPath == "" {
		return
	}

	routeFile := findRouteZipFile(reader.File, refPath)
	if routeFile == nil {
		return
	}

	opened, err := routeFile.Open()
	if err != nil {
		return
	}
	defer opened.Close()

	points, err := parseGPXPoints(opened)
	if err != nil {
		return
	}
	activity.RoutePoints = points
}

// findRouteZipFile matches a WorkoutRoute FileReference path (which carries
// a leading slash, e.g. "/workout-routes/route_x.gpx") against the zip's
// entries by suffix, so it works regardless of the export directory prefix
// (e.g. "apple_health_export/workout-routes/route_x.gpx").
func findRouteZipFile(files []*zip.File, refPath string) *zip.File {
	trimmed := strings.TrimPrefix(refPath, "/")
	if trimmed == "" {
		return nil
	}
	for _, file := range files {
		if strings.HasSuffix(file.Name, trimmed) {
			return file
		}
	}
	return nil
}

// decodeAppleWorkouts streams through an export.xml document, decoding each
// <Workout> element's full subtree while skipping everything else (notably
// the potentially millions of sibling <Record> elements). It must remain
// streaming rather than reading the whole document into memory.
func decodeAppleWorkouts(r io.Reader) ([]ParsedActivity, error) {
	workouts, err := decodeAppleWorkoutsRaw(r)
	if err != nil {
		return nil, err
	}

	var activities []ParsedActivity
	for _, workout := range workouts {
		activity, ok := workoutToActivity(workout)
		if ok {
			activities = append(activities, activity)
		}
	}
	return activities, nil
}

// decodeAppleWorkoutsRaw performs the streaming token walk over an
// export.xml document and returns the decoded appleWorkout structs
// (including their nested subtrees) without converting them to
// ParsedActivity. Exposed separately so tests can assert on the captured
// nested data directly.
func decodeAppleWorkoutsRaw(r io.Reader) ([]appleWorkout, error) {
	decoder := xml.NewDecoder(r)
	var workouts []appleWorkout
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		start, ok := token.(xml.StartElement)
		if !ok || start.Name.Local != "Workout" {
			continue
		}

		var workout appleWorkout
		if err := decoder.DecodeElement(&workout, &start); err != nil {
			return nil, err
		}
		workouts = append(workouts, workout)
	}
	return workouts, nil
}

func workoutToActivity(workout appleWorkout) (ParsedActivity, bool) {
	startedAt, err := parseAppleTime(workout.StartDate)
	if err != nil {
		return ParsedActivity{}, false
	}
	var endedAt *time.Time
	if parsed, err := parseAppleTime(workout.EndDate); err == nil {
		endedAt = &parsed
	}

	durationS := parseDurationSeconds(workout.Duration, workout.DurationUnit)
	distanceM := parseDistanceMeters(workout.TotalDistance, workout.TotalDistanceUnit)
	externalID := stableWorkoutSourceKey(workout)
	contentHash := workoutContentHash(workout)

	return ParsedActivity{
		Provider:         "apple_health",
		ExternalID:       externalID,
		SportType:        normalizeAppleSport(workout.WorkoutActivityType),
		Title:            strings.TrimPrefix(workout.WorkoutActivityType, "HKWorkoutActivityType"),
		StartedAt:        startedAt,
		EndedAt:          endedAt,
		DistanceM:        distanceM,
		DurationS:        durationS,
		SourceKind:       "apple_health_export",
		SourceActivityID: externalID,
		ContentHash:      contentHash,
	}, true
}

// stableWorkoutSourceKey builds an identity for an Apple Health workout that
// is stable ACROSS full-export re-exports of the same underlying workout.
// It deliberately does NOT include the export zip's sha256 (that changes on
// every export even when the workout itself hasn't). It combines the
// workout's source name, a normalized device fingerprint (see
// stableDeviceKey), activity type, start/end timestamps and duration -
// fields that together uniquely and durably identify a single workout
// recorded by a single device.
func stableWorkoutSourceKey(workout appleWorkout) string {
	return strings.Join([]string{
		workout.SourceName,
		stableDeviceKey(workout.Device),
		workout.WorkoutActivityType,
		workout.StartDate,
		workout.EndDate,
		workout.Duration,
	}, "|")
}

// deviceFieldRe extracts "name:" and "hardware:" values out of Apple's
// HKDevice description string, e.g.:
//
//	<<HKDevice: 0x8dcef18c0>, name:Apple Watch, manufacturer:Apple Inc.,
//	  model:Watch, hardware:Watch7,1, software:10.2, creation date:...>>
//
// The leading "0x..." token is a runtime pointer and "creation date:" is
// wall-clock capture time - both are volatile and differ between exports of
// the SAME physical device. Only "name" and "hardware" are stable, so those
// are the only fields pulled out.
var deviceFieldRe = regexp.MustCompile(`(?:^|,\s*)(name|hardware):([^,>]*)`)

// stableDeviceKey normalizes a raw Apple HKDevice description string into a
// fingerprint that is stable across exports of the same device, by
// extracting only the "name" and "hardware" fields and dropping the
// volatile pointer address and creation-date segment.
func stableDeviceKey(device string) string {
	if device == "" {
		return ""
	}
	matches := deviceFieldRe.FindAllStringSubmatch(device, -1)
	if len(matches) == 0 {
		return ""
	}
	values := make(map[string]string, len(matches))
	for _, m := range matches {
		values[m[1]] = strings.TrimSpace(m[2])
	}
	return fmt.Sprintf("name=%s;hardware=%s", values["name"], values["hardware"])
}

// workoutContentHash computes a sha256 hex digest over the change-relevant
// fields of a workout: its own attrs (with device normalized via
// stableDeviceKey, since the raw device string carries a volatile pointer
// and creation date that would otherwise make every re-export look
// "changed"), its WorkoutStatistics entries in document order, and its
// workout-level MetadataEntry values deduped by key (real exports can
// repeat metadata keys under one Workout) and sorted for determinism.
func workoutContentHash(workout appleWorkout) string {
	var b strings.Builder

	fmt.Fprintf(&b, "type=%s\n", workout.WorkoutActivityType)
	fmt.Fprintf(&b, "start=%s\n", workout.StartDate)
	fmt.Fprintf(&b, "end=%s\n", workout.EndDate)
	fmt.Fprintf(&b, "duration=%s\n", workout.Duration)
	fmt.Fprintf(&b, "durationUnit=%s\n", workout.DurationUnit)
	fmt.Fprintf(&b, "totalDistance=%s\n", workout.TotalDistance)
	fmt.Fprintf(&b, "totalDistanceUnit=%s\n", workout.TotalDistanceUnit)
	fmt.Fprintf(&b, "sourceName=%s\n", workout.SourceName)
	fmt.Fprintf(&b, "sourceVersion=%s\n", workout.SourceVersion)
	fmt.Fprintf(&b, "device=%s\n", stableDeviceKey(workout.Device))

	if workout.WorkoutRoute != nil && workout.WorkoutRoute.FileReference != nil {
		fmt.Fprintf(&b, "routePath=%s\n", workout.WorkoutRoute.FileReference.Path)
	}
	fmt.Fprintf(&b, "routeSyncId=%s\n", workout.WorkoutRoute.syncIdentifier())

	for _, stat := range workout.Statistics {
		fmt.Fprintf(&b, "stat:type=%s|sum=%s|avg=%s|max=%s|min=%s|unit=%s|start=%s|end=%s\n",
			stat.Type, stat.Sum, stat.Average, stat.Maximum, stat.Minimum, stat.Unit, stat.StartDate, stat.EndDate)
	}

	metadata := make(map[string]string, len(workout.MetadataEntries))
	for _, entry := range workout.MetadataEntries {
		metadata[entry.Key] = entry.Value
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "meta:%s=%s\n", key, metadata[key])
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func parseAppleTime(value string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05 -0700", value)
}

func parseDurationSeconds(value string, unit string) *int {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	switch unit {
	case "min":
		parsed *= 60
	case "hr":
		parsed *= 3600
	}
	seconds := int(parsed)
	return &seconds
}

func parseDistanceMeters(value string, unit string) *float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	switch unit {
	case "km":
		parsed *= 1000
	case "mi":
		parsed *= 1609.344
	}
	return &parsed
}

func normalizeAppleSport(value string) string {
	switch value {
	case "HKWorkoutActivityTypeRunning":
		return "run"
	case "HKWorkoutActivityTypeWalking":
		return "walk"
	case "HKWorkoutActivityTypeCycling":
		return "ride"
	case "HKWorkoutActivityTypeHiking":
		return "hike"
	default:
		return strings.TrimPrefix(value, "HKWorkoutActivityType")
	}
}
