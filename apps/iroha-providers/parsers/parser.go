package parsers

import (
	"fmt"
	"path/filepath"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
)

type Input struct {
	ParserKind       string
	StoragePath      string
	OriginalFilename string
	RawFileSHA256    string
}

type ActivityObservation = observations.Activity

// ActivitySampling is a single timestamped sensor sample (e.g. an Apple
// Health <Record>) attached to a ActivityObservation because its timestamp falls
// inside that activity's [StartedAt, EndedAt] window.
type ActivitySampling = observations.Sampling

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

type SleepSegmentObservation = observations.SleepSegment

type SleepObservation = observations.Sleep

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

type DailySummaryObservation = observations.DailySummary

type DailyMetricObservation = observations.DailyMetric

type RoutePoint = observations.RoutePoint

// ActivityLap is a timing-only lap span derived from lap-boundary events.
// Distance/HR/pace per lap require per-sample data (task-7) and are
// deliberately left out here.
type ActivityLap = observations.Lap

// Kind* are the parser kinds this package actually implements. They are the
// single source of truth for what an import may request: the API rejects any
// other parser_kind up front (see IsImplemented) rather than enqueuing a job
// that can only fail. New formats (fit, tcx, strava_export) get a Kind* const
// and a Parse case here once their parser lands.
const (
	KindGPX               = coreimports.KindGPX
	KindAppleHealthExport = coreimports.KindAppleHealthExport
	KindAniList           = coreimports.KindAniList
	KindBangumi           = coreimports.KindBangumi
)

// IsImplemented reports whether Parse has a working parser for kind.
func IsImplemented(kind string) bool {
	switch kind {
	case KindGPX, KindAppleHealthExport, KindAniList, KindBangumi:
		return true
	default:
		return false
	}
}

func Parse(input Input) ([]ActivityObservation, error) {
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
