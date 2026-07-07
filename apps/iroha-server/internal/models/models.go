package models

import (
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
	return "raw_files"
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
	return "import_jobs"
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
	SourceKind       string
	SourceActivityID string
	FirstRawFileID   uuid.UUID `gorm:"type:uuid"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (Activity) TableName() string {
	return "activities"
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
	return "external_refs"
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
	return "activity_route_points"
}
