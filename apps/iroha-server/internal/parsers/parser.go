package parsers

import (
	"fmt"
	"path/filepath"
	"time"
)

type Input struct {
	ParserKind       string
	StoragePath      string
	OriginalFilename string
	RawFileSHA256    string
}

type ParsedActivity struct {
	Provider         string
	ExternalID       string
	SportType        string
	Title            string
	StartedAt        time.Time
	EndedAt          *time.Time
	DistanceM        *float64
	DurationS        *int
	SourceKind       string
	SourceActivityID string
	// ContentHash is a sha256 hex digest over the change-relevant fields of
	// the source record, used to detect whether a re-export produced the
	// same content for an already-known source item. Non-Apple parsers
	// leave this empty, which opts the activity out of change-detection
	// (it always follows the plain upsert path).
	ContentHash string
	RoutePoints []RoutePoint

	// AvgHR, MaxHR and AvgPaceSPerKM are derived from the source's
	// per-workout statistics (e.g. Apple Health WorkoutStatistics), not
	// computed from route/sample data. Nil when the source has no such
	// statistic.
	AvgHR         *int
	MaxHR         *int
	AvgPaceSPerKM *float64

	// Laps are derived from lap-delimiting events in the source (e.g. Apple
	// Health WorkoutEvent entries of type Lap/Segment). Nil when the source
	// has no such events - most workouts won't have them.
	Laps []ParsedLap
}

type RoutePoint struct {
	Ts         *time.Time
	Lat        float64
	Lon        float64
	ElevationM *float64
}

// ParsedLap is a timing-only lap span derived from lap-boundary events.
// Distance/HR/pace per lap require per-sample data (task-7) and are
// deliberately left out here.
type ParsedLap struct {
	LapNo     int
	StartTs   *time.Time
	EndTs     *time.Time
	DurationS *int
}

func Parse(input Input) ([]ParsedActivity, error) {
	switch input.ParserKind {
	case "gpx":
		return ParseGPXFile(input.StoragePath, GPXOptions{
			Title:      titleFromFilename(input.OriginalFilename),
			ExternalID: input.RawFileSHA256,
		})
	case "apple_health_export":
		return ParseAppleHealthExport(input.StoragePath, input.RawFileSHA256)
	default:
		return nil, fmt.Errorf("parser %q is not implemented yet", input.ParserKind)
	}
}

func titleFromFilename(filename string) string {
	base := filepath.Base(filename)
	ext := filepath.Ext(base)
	if ext != "" {
		base = base[:len(base)-len(ext)]
	}
	if base == "" || base == "." {
		return "Imported activity"
	}
	return base
}
