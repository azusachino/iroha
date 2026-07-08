package parsers

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	FileReference *appleFileReference `xml:"FileReference"`
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
		if strings.HasSuffix(file.Name, "export.xml") {
			parsed, err := parseAppleWorkouts(file, rawHash)
			if err != nil {
				return nil, err
			}
			activities = append(activities, parsed...)
		}
		if strings.Contains(file.Name, "workout-routes/") && strings.HasSuffix(strings.ToLower(file.Name), ".gpx") {
			parsed, err := parseZippedGPX(file, rawHash)
			if err != nil {
				return nil, err
			}
			activities = append(activities, parsed...)
		}
	}
	return activities, nil
}

func parseAppleWorkouts(file *zip.File, rawHash string) ([]ParsedActivity, error) {
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	return decodeAppleWorkouts(opened, rawHash)
}

// decodeAppleWorkouts streams through an export.xml document, decoding each
// <Workout> element's full subtree while skipping everything else (notably
// the potentially millions of sibling <Record> elements). It must remain
// streaming rather than reading the whole document into memory.
func decodeAppleWorkouts(r io.Reader, rawHash string) ([]ParsedActivity, error) {
	workouts, err := decodeAppleWorkoutsRaw(r)
	if err != nil {
		return nil, err
	}

	var activities []ParsedActivity
	for _, workout := range workouts {
		activity, ok := workoutToActivity(workout, rawHash)
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

func parseZippedGPX(file *zip.File, rawHash string) ([]ParsedActivity, error) {
	opened, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer opened.Close()

	temp, err := os.CreateTemp("", "iroha-route-*.gpx")
	if err != nil {
		return nil, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)

	if _, err := io.Copy(temp, opened); err != nil {
		temp.Close()
		return nil, err
	}
	if err := temp.Close(); err != nil {
		return nil, err
	}

	return ParseGPXFile(tempPath, GPXOptions{
		Title:      titleFromFilename(filepath.Base(file.Name)),
		ExternalID: rawHash + ":" + file.Name,
	})
}

func workoutToActivity(workout appleWorkout, rawHash string) (ParsedActivity, bool) {
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
	externalID := fmt.Sprintf("%s:%s:%s", rawHash, workout.WorkoutActivityType, workout.StartDate)

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
	}, true
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
