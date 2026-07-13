package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type RawFile struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	SHA256           string    `gorm:"column:sha256"`
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
	StoragePath      string
	SourceKind       string
	UploadedVia      string
	CreatedAt        time.Time
}

func (RawFile) TableName() string {
	return "tb_raw_files"
}

type ImportJob struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	RawFileID     uuid.UUID `gorm:"type:uuid"`
	Status        string
	ParserKind    string
	ParserVersion string
	ErrorMessage  *string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	CreatedAt     time.Time
}

func (ImportJob) TableName() string {
	return "tb_import_jobs"
}

type Activity struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	SportType             string
	Title                 string
	StartedAt             time.Time
	EndedAt               *time.Time
	Timezone              string
	DistanceM             *float64
	DurationS             *int
	MovingTimeS           *int
	ElevationGainM        *float64
	AvgHR                 *int
	MaxHR                 *int
	AvgPaceSPerKM         *float64
	CaloriesKcal          *float64
	SourceKind            string
	SourceActivityID      string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (Activity) TableName() string {
	return "tb_activities"
}

type SourceObservation struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey"`
	Provider            string
	SourceKind          string
	SourceKey           string
	ContentHash         string
	RawFileID           uuid.UUID  `gorm:"type:uuid"`
	FirstSeenSnapshotID *uuid.UUID `gorm:"type:uuid"`
	LastSeenSnapshotID  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (SourceObservation) TableName() string { return "tb_source_observations" }

type ActivityObservation struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID       uuid.UUID `gorm:"type:uuid"`
	SourceActivityID string
	SportType        string
	Title            string
	StartedAt        time.Time
	EndedAt          *time.Time
	DistanceM        *float64
	DurationS        *int
	MovingTimeS      *int
	ElevationGainM   *float64
	AvgHR            *int
	MaxHR            *int
	AvgPaceSPerKM    *float64
	CaloriesKcal     *float64
	MatchStatus      string
	MatchConfidence  *float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (ActivityObservation) TableName() string { return "tb_activity_observations" }

type ExternalRef struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID uuid.UUID `gorm:"type:uuid"`
	Provider   string
	ExternalID string
	RawFileID  uuid.UUID `gorm:"type:uuid"`
	CreatedAt  time.Time
}

func (ExternalRef) TableName() string {
	return "tb_external_refs"
}

type ActivityRoutePoint struct {
	ActivityID uuid.UUID `gorm:"type:uuid;primaryKey"`
	Seq        int       `gorm:"primaryKey"`
	Ts         *time.Time
	Lat        float64
	Lon        float64
	ElevationM *float64
	DistanceM  *float64
	SpeedMPS   *float64
	HeartRate  *int
}

func (ActivityRoutePoint) TableName() string {
	return "tb_activity_route_points"
}

type ActivitySampling struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID   uuid.UUID `gorm:"type:uuid"`
	SamplingType string
	Ts           time.Time
	Value        float64
	Unit         string
}

func (ActivitySampling) TableName() string {
	return "tb_activity_samplings"
}

type ActivityLap struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ActivityID    uuid.UUID `gorm:"type:uuid"`
	LapNo         int
	StartTs       *time.Time
	EndTs         *time.Time
	DistanceM     *float64
	DurationS     *int
	AvgHR         *int
	AvgPaceSPerKM *float64
	CaloriesKcal  *float64
}

func (ActivityLap) TableName() string {
	return "tb_activity_laps"
}

type SleepSession struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	WakeDate              time.Time `gorm:"type:date"`
	StartedAt             time.Time
	EndedAt               time.Time
	TimeInBedS            int
	AsleepS               int
	Efficiency            float64
	IsMainSleep           bool
	CoreS                 int
	DeepS                 int
	RemS                  int
	AwakeS                int
	UnspecifiedS          int
	Source                string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (SleepSession) TableName() string {
	return "tb_sleep_sessions"
}

type SleepObservation struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	SleepSessionID  uuid.UUID `gorm:"type:uuid"`
	WakeDate        time.Time `gorm:"type:date"`
	StartedAt       time.Time
	EndedAt         time.Time
	TimeInBedS      int
	AsleepS         int
	Efficiency      float64
	IsMainSleep     bool
	CoreS           int
	DeepS           int
	RemS            int
	AwakeS          int
	UnspecifiedS    int
	Source          string
	MatchStatus     string
	MatchConfidence *float64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (SleepObservation) TableName() string { return "tb_sleep_observations" }

type SleepSegment struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	SessionID uuid.UUID `gorm:"type:uuid"`
	Stage     string
	StartedAt time.Time
	EndedAt   time.Time
	Seq       int
}

func (SleepSegment) TableName() string {
	return "tb_sleep_segments"
}

type DailySummary struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Day                   time.Time `gorm:"type:date"`
	MoveKcal              float64
	MoveGoalKcal          float64
	ExerciseMin           float64
	ExerciseGoalMin       float64
	StandHours            float64
	StandGoalHours        float64
	Source                string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (DailySummary) TableName() string {
	return "tb_daily_summaries"
}

type DailySummaryObservation struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	DailySummaryID  uuid.UUID `gorm:"type:uuid"`
	Day             time.Time `gorm:"type:date"`
	MoveKcal        float64
	MoveGoalKcal    float64
	ExerciseMin     float64
	ExerciseGoalMin float64
	StandHours      float64
	StandGoalHours  float64
	Source          string
	MatchStatus     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (DailySummaryObservation) TableName() string { return "tb_daily_summary_observations" }

type DailyMetric struct {
	ID                    uuid.UUID `gorm:"type:uuid;primaryKey"`
	Day                   time.Time `gorm:"type:date"`
	Metric                string
	Value                 float64
	Unit                  string
	Source                string
	FirstRawFileID        uuid.UUID `gorm:"type:uuid"`
	CreatedAt             time.Time
	UpdatedAt             time.Time
	SelectedObservationID *uuid.UUID `gorm:"type:uuid"`
}

func (DailyMetric) TableName() string {
	return "tb_daily_metrics"
}

type DailyMetricObservation struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	DailyMetricID uuid.UUID `gorm:"type:uuid"`
	Day           time.Time `gorm:"type:date"`
	Metric        string
	Value         float64
	Unit          string
	Source        string
	Reducer       string
	MatchStatus   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (DailyMetricObservation) TableName() string { return "tb_daily_metric_observations" }

type ImportSnapshot struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	ImportJobID   uuid.UUID `gorm:"type:uuid"`
	RawFileID     uuid.UUID `gorm:"type:uuid"`
	SHA256        string    `gorm:"column:sha256"`
	ParserVersion string
	TakenAt       *time.Time
	CreatedAt     time.Time
}

func (ImportSnapshot) TableName() string {
	return "tb_import_snapshots"
}

type AppleSourceItem struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	SourceKey          string
	ItemType           string
	ContentHash        string
	ActivityID         *uuid.UUID `gorm:"type:uuid"`
	SleepSessionID     *uuid.UUID `gorm:"type:uuid"`
	DailySummaryID     *uuid.UUID `gorm:"type:uuid"`
	DailyMetricID      *uuid.UUID `gorm:"type:uuid"`
	LastSeenSnapshotID *uuid.UUID `gorm:"type:uuid"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (AppleSourceItem) TableName() string {
	return "tb_apple_source_items"
}

type IntakePayload struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey"`
	SourceKind    string
	SourceActor   string
	SourceEventID string
	ContentType   string
	SHA256        string `gorm:"column:sha256"`
	SizeBytes     int64
	StoragePath   string
	PayloadJSON   *json.RawMessage `gorm:"column:payload_json;type:jsonb"`
	ReceivedAt    time.Time
	ParsedAt      *time.Time
	CreatedAt     time.Time
}

func (IntakePayload) TableName() string {
	return "tb_intake_payloads"
}

type Job struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Kind         string
	Status       string
	Priority     int
	PayloadJSON  json.RawMessage `gorm:"column:payload_json;type:jsonb"`
	Attempts     int
	MaxAttempts  int
	RunAfter     time.Time
	LockedBy     *string
	LockedAt     *time.Time
	ErrorMessage *string
	StartedAt    *time.Time
	FinishedAt   *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (Job) TableName() string {
	return "tb_jobs"
}

type JobSchedule struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Kind         string
	Enabled      bool
	ScheduleKind string
	ScheduleExpr string
	PayloadJSON  json.RawMessage `gorm:"column:payload_json;type:jsonb"`
	NextRunAt    *time.Time
	LastRunAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (JobSchedule) TableName() string {
	return "tb_job_schedules"
}
