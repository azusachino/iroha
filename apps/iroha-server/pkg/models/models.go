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
	ID               uuid.UUID `gorm:"type:uuid;primaryKey"`
	SportType        string
	Title            string
	StartedAt        time.Time
	EndedAt          *time.Time
	Timezone         string
	DistanceM        *float64
	DurationS        *int
	MovingTimeS      *int
	ElevationGainM   *float64
	AvgHR            *int
	MaxHR            *int
	AvgPaceSPerKM    *float64
	CaloriesKcal     *float64
	SourceKind       string
	SourceActivityID string
	FirstRawFileID   uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Activity) TableName() string {
	return "tb_activities"
}

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
	ID             uuid.UUID `gorm:"type:uuid;primaryKey"`
	WakeDate       time.Time `gorm:"type:date"`
	StartedAt      time.Time
	EndedAt        time.Time
	TimeInBedS     int
	AsleepS        int
	Efficiency     float64
	IsMainSleep    bool
	CoreS          int
	DeepS          int
	RemS           int
	AwakeS         int
	UnspecifiedS   int
	Source         string
	FirstRawFileID uuid.UUID `gorm:"type:uuid"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (SleepSession) TableName() string {
	return "tb_sleep_sessions"
}

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
