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
	CaloriesKcal  *float64

	// Laps are derived from lap-delimiting events in the source (e.g. Apple
	// Health WorkoutEvent entries of type Lap/Segment). Nil when the source
	// has no such events - most workouts won't have them.
	Laps []ParsedLap

	// Samplings are per-timestamp sensor samples (heart rate, running
	// power, etc.) associated with this activity via its time window (see
	// apple_health.go's pass-2 record stream). Nil when the source has no
	// selected-type records falling inside the activity's window.
	Samplings []ParsedSampling
}

// ParsedSampling is a single timestamped sensor sample (e.g. an Apple
// Health <Record>) attached to a ParsedActivity because its timestamp falls
// inside that activity's [StartedAt, EndedAt] window.
type ParsedSampling struct {
	SamplingType string
	Ts           time.Time
	Value        float64
	Unit         string
}

const (
	SleepStageInBed             = "in_bed"
	SleepStageAwake             = "awake"
	SleepStageCore              = "core"
	SleepStageDeep              = "deep"
	SleepStageREM               = "rem"
	SleepStageAsleepUnspecified = "asleep_unspecified"

	DefaultSleepSessionGap = time.Hour
	MainSleepThreshold     = 3 * time.Hour
)

type ParsedSleepSegment struct {
	Stage     string
	StartedAt time.Time
	EndedAt   time.Time
	Source    string
}

type ParsedSleepSession struct {
	WakeDate     time.Time
	StartedAt    time.Time
	EndedAt      time.Time
	TimeInBedS   int
	AsleepS      int
	Efficiency   float64
	IsMainSleep  bool
	CoreS        int
	DeepS        int
	RemS         int
	AwakeS       int
	UnspecifiedS int
	Source       string
	Segments     []ParsedSleepSegment
}

const (
	DailyMetricSteps           = "steps"
	DailyMetricDistanceKM      = "distance_km"
	DailyMetricFlights         = "flights"
	DailyMetricRestingHR       = "resting_hr"
	DailyMetricWalkingHR       = "walking_hr_avg"
	DailyMetricHRVSDNN         = "hrv_sdnn"
	DailyMetricVO2Max          = "vo2max"
	DailyMetricBodyMassKG      = "body_mass_kg"
	DailyMetricSpO2Avg         = "spo2_avg"
	DailyMetricSpO2Min         = "spo2_min"
	DailyMetricRespiratoryRate = "respiratory_rate"
)

type ParsedDailySummary struct {
	Day             time.Time
	MoveKcal        float64
	MoveGoalKcal    float64
	ExerciseMin     float64
	ExerciseGoalMin float64
	StandHours      float64
	StandGoalHours  float64
	Source          string
}

type ParsedDailyMetric struct {
	Day    time.Time
	Metric string
	Value  float64
	Unit   string
	Source string
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
	LapNo        int
	StartTs      *time.Time
	EndTs        *time.Time
	DurationS    *int
	CaloriesKcal *float64
}

// Kind* are the parser kinds this package actually implements. They are the
// single source of truth for what an import may request: the API rejects any
// other parser_kind up front (see IsImplemented) rather than enqueuing a job
// that can only fail. New formats (fit, tcx, strava_export) get a Kind* const
// and a Parse case here once their parser lands.
const (
	KindGPX               = "gpx"
	KindAppleHealthExport = "apple_health_export"
)

// IsImplemented reports whether Parse has a working parser for kind.
func IsImplemented(kind string) bool {
	switch kind {
	case KindGPX, KindAppleHealthExport:
		return true
	default:
		return false
	}
}

func Parse(input Input) ([]ParsedActivity, error) {
	switch input.ParserKind {
	case KindGPX:
		return ParseGPXFile(input.StoragePath, GPXOptions{
			Title:      titleFromFilename(input.OriginalFilename),
			ExternalID: input.RawFileSHA256,
		})
	case KindAppleHealthExport:
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
