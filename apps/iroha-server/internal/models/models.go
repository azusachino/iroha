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
