package imports

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/internal/ids"
	"github.com/azusachino/iroha/apps/iroha-server/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusQueued    = "queued"
	StatusParsing   = "parsing"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

type Service struct {
	db            *gorm.DB
	logger        *slog.Logger
	parserVersion string
}

type CreateInput struct {
	RawFileID  string
	ParserKind string
}

func NewService(db *gorm.DB, logger *slog.Logger, parserVersion string) *Service {
	if parserVersion == "" {
		parserVersion = "dev"
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, logger: logger, parserVersion: parserVersion}
}

func (s *Service) Create(input CreateInput) (models.ImportJob, error) {
	rawFileID, err := ids.Decode(ids.RawFilePrefix, input.RawFileID)
	if err != nil {
		return models.ImportJob{}, err
	}

	if err := s.ensureRawFileExists(rawFileID); err != nil {
		return models.ImportJob{}, err
	}

	id, err := ids.New()
	if err != nil {
		return models.ImportJob{}, err
	}

	job := models.ImportJob{
		ID:            id,
		RawFileID:     rawFileID,
		Status:        StatusQueued,
		ParserKind:    input.ParserKind,
		ParserVersion: s.parserVersion,
		CreatedAt:     time.Now().UTC(),
	}
	if err := s.db.Create(&job).Error; err != nil {
		return models.ImportJob{}, err
	}

	go s.ProcessAsync(job.ID)
	return job, nil
}

func (s *Service) List(limit int) ([]models.ImportJob, error) {
	var jobs []models.ImportJob
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	err := s.db.Order("created_at desc").Limit(limit).Find(&jobs).Error
	return jobs, err
}

func (s *Service) Get(id string) (models.ImportJob, bool, error) {
	decoded, err := ids.Decode(ids.ImportPrefix, id)
	if err != nil {
		return models.ImportJob{}, false, err
	}
	return s.getByUUID(decoded)
}

func (s *Service) ProcessAsync(jobID uuid.UUID) {
	if err := s.process(jobID); err != nil {
		s.logger.Error("process import job", "job_id", jobID.String(), "error", err)
	}
}

func (s *Service) process(jobID uuid.UUID) error {
	now := time.Now().UTC()
	if err := s.db.Model(&models.ImportJob{}).
		Where("id = ? and status = ?", jobID, StatusQueued).
		Updates(map[string]any{
			"status":     StatusParsing,
			"started_at": &now,
		}).Error; err != nil {
		return err
	}

	job, found, err := s.getByUUID(jobID)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("import job not found")
	}

	finishedAt := time.Now().UTC()
	errorMessage := fmt.Sprintf("parser %q is not implemented yet", job.ParserKind)
	return s.db.Model(&models.ImportJob{}).
		Where("id = ?", jobID).
		Updates(map[string]any{
			"status":        StatusFailed,
			"error_message": &errorMessage,
			"finished_at":   &finishedAt,
		}).Error
}

func (s *Service) getByUUID(id uuid.UUID) (models.ImportJob, bool, error) {
	var job models.ImportJob
	err := s.db.First(&job, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ImportJob{}, false, nil
	}
	if err != nil {
		return models.ImportJob{}, false, err
	}
	return job, true, nil
}

func (s *Service) ensureRawFileExists(id uuid.UUID) error {
	var count int64
	if err := s.db.Model(&models.RawFile{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("raw_file not found")
	}
	return nil
}
