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
