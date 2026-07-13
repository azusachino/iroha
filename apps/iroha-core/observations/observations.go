// Package observations contains provider-neutral contracts shared by the
// HTTP server and background importer. It deliberately has no database,
// transport, or provider implementation dependencies.
package observations

import (
	"time"

	"github.com/google/uuid"
)

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
	Status           string
	Progress         *float64
	Score            *float64
	StartedAt        *time.Time
	CompletedAt      *time.Time
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
	EventAt         *time.Time
	SourceEventID   string
	Unit            string
	Position        *float64
	Total           *float64
	ProgressPercent *float64
	Rating          *float64
	RatingScale     *float64
	Note            string
}

type MediaProgress struct {
	Status             string
	Unit               string
	Position           *float64
	Total              *float64
	ProgressPercent    *float64
	StartedAt          *time.Time
	LastUpdateAt       *time.Time
	FinishedAt         *time.Time
	PlayCount          int
	HiddenFromContinue bool
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
