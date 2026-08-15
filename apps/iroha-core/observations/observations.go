// Package observations contains provider-neutral contracts shared by the
// HTTP server and background importer. It deliberately has no database,
// transport, or provider implementation dependencies.
package observations

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// DatePrecision records how much of a calendar date a source actually knows.
// Value is always the first day of the represented period for storage and
// comparison; callers must use Precision when rendering it.
type DatePrecision string

const (
	DatePrecisionYear  DatePrecision = "year"
	DatePrecisionMonth DatePrecision = "month"
	DatePrecisionDay   DatePrecision = "day"
)

type PartialDate struct {
	Value     time.Time
	Precision DatePrecision
}

func (d PartialDate) Valid() bool {
	if d.Value.IsZero() {
		return false
	}
	switch d.Precision {
	case DatePrecisionYear:
		return d.Value.Month() == time.January && d.Value.Day() == 1
	case DatePrecisionMonth:
		return d.Value.Day() == 1
	case DatePrecisionDay:
		return true
	default:
		return false
	}
}

func (d PartialDate) String() string {
	if !d.Valid() {
		return ""
	}
	switch d.Precision {
	case DatePrecisionYear:
		return d.Value.Format("2006")
	case DatePrecisionMonth:
		return d.Value.Format("2006-01")
	case DatePrecisionDay:
		return d.Value.Format("2006-01-02")
	default:
		return ""
	}
}

func NewPartialDate(year int, month, day int) (*PartialDate, error) {
	if year < 1 || year > 9999 {
		return nil, fmt.Errorf("invalid partial date year %d", year)
	}
	precision := DatePrecisionYear
	if month != 0 {
		if month < 1 || month > 12 {
			return nil, fmt.Errorf("invalid partial date month %d", month)
		}
		precision = DatePrecisionMonth
	}
	if day != 0 {
		if month == 0 || day < 1 || day > daysInMonth(year, time.Month(month)) {
			return nil, fmt.Errorf("invalid partial date day %d", day)
		}
		precision = DatePrecisionDay
	}
	return &PartialDate{Value: time.Date(year, time.Month(maxInt(month, 1)), maxInt(day, 1), 0, 0, 0, 0, time.UTC), Precision: precision}, nil
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func maxInt(value, fallback int) int {
	if value < fallback {
		return fallback
	}
	return value
}

type Activity struct {
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
	ContentHash      string
	RoutePoints      []RoutePoint
	AvgHR            *int
	MaxHR            *int
	AvgPaceSPerKM    *float64
	CaloriesKcal     *float64
	Laps             []Lap
	Samplings        []Sampling
}

type Sampling struct {
	SamplingType string
	Ts           time.Time
	Value        float64
	Unit         string
}

type SleepSegment struct {
	Stage     string
	StartedAt time.Time
	EndedAt   time.Time
	Source    string
}

type Sleep struct {
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
	Segments     []SleepSegment
}

type DailySummary struct {
	Day             time.Time
	MoveKcal        float64
	MoveGoalKcal    float64
	ExerciseMin     float64
	ExerciseGoalMin float64
	StandHours      float64
	StandGoalHours  float64
	Source          string
}

type DailyMetric struct {
	Day    time.Time
	Metric string
	Value  float64
	Unit   string
	Source string
}

type Media struct {
	Provider         string
	ExternalID       string
	MediaType        string
	ItemRole         string
	Title            string
	WorkExternalID   string
	WorkKind         string
	WorkTitle        string
	ParentExternalID string
	ReleaseDate      *time.Time
	ReleaseDateOn    *PartialDate
	SeasonNumber     *int
	EpisodeNumber    *int
	ChapterNumber    *float64
	VolumeNumber     *float64
	DurationSeconds  *int
	PageCount        *int
	EpisodeCount     *int
	ChapterCount     *int
	Language         string
	Country          string
	CoverImageURL    string
	Description      string
	Status           string
	Progress         *float64
	Score            *float64
	StartedOn        *PartialDate
	CompletedOn      *PartialDate
	StateSourceID    string
	StateNote        string
	StateRatingScale *float64
	Titles           []MediaTitle
	ExternalRefs     []MediaExternalRef
	Relations        []MediaRelation
	Events           []MediaEvent
	ProgressState    *MediaProgress
}

type MediaTitle struct {
	Title      string
	Language   string
	Script     string
	Region     string
	TitleKind  string
	Provider   string
	IsPrimary  bool
	Confidence *float64
}

type MediaExternalRef struct {
	Provider    string
	ExternalID  string
	ExternalURL string
	MatchedBy   string
	Confidence  *float64
}

type MediaRelation struct {
	FromType       string
	FromExternalID string
	ToType         string
	ToExternalID   string
	RelationType   string
	Provider       string
	Confidence     *float64
}

type MediaEvent struct {
	EventType       string
	EventAt         time.Time
	SourceEventID   string
	Unit            string
	Position        *float64
	Total           *float64
	ProgressPercent *float64
	Rating          *float64
	RatingScale     *float64
	Note            string
}

const (
	MediaEventStarted    = "started"
	MediaEventProgressed = "progressed"
	MediaEventCompleted  = "completed"
	MediaEventFinished   = "finished"
	MediaEventRead       = "read"
	MediaEventWatched    = "watched"
	MediaEventListened   = "listened"
	MediaEventReread     = "reread"
	MediaEventRewatched  = "rewatched"
	MediaEventAbandoned  = "abandoned"
	MediaEventPaused     = "paused"
	MediaEventReopened   = "reopened"
	MediaEventRated      = "rated"
	MediaEventNoted      = "noted"
	MediaEventBookmarked = "bookmarked"
)

var mediaEventTypes = map[string]struct{}{
	MediaEventStarted: {}, MediaEventProgressed: {}, MediaEventCompleted: {}, MediaEventFinished: {},
	MediaEventRead: {}, MediaEventWatched: {}, MediaEventListened: {}, MediaEventReread: {},
	MediaEventRewatched: {}, MediaEventAbandoned: {}, MediaEventPaused: {}, MediaEventReopened: {},
	MediaEventRated: {}, MediaEventNoted: {}, MediaEventBookmarked: {},
}

func ValidMediaEventType(value string) bool {
	_, ok := mediaEventTypes[value]
	return ok
}

type MediaProgress struct {
	Status             string
	Unit               string
	Position           *float64
	Total              *float64
	ProgressPercent    *float64
	LastUpdateAt       *time.Time
	StartedOn          *PartialDate
	CompletedOn        *PartialDate
	PlayCount          int
	HiddenFromContinue bool
	Paused             bool
}

type RoutePoint struct {
	Ts         *time.Time
	Lat        float64
	Lon        float64
	ElevationM *float64
}

type Lap struct {
	LapNo        int
	StartTs      *time.Time
	EndTs        *time.Time
	DurationS    *int
	CaloriesKcal *float64
}

type SourceIdentity struct {
	Provider   string
	SourceKind string
	SourceKey  string
}

type SourceObservation struct {
	ID          uuid.UUID
	Identity    SourceIdentity
	ContentHash string
	RawFileID   uuid.UUID
	SnapshotID  uuid.UUID
}
