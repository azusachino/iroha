package imports

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/internal/ids"
	"github.com/azusachino/iroha/apps/iroha-server/internal/models"
	"github.com/azusachino/iroha/apps/iroha-server/internal/parsers"
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

	rawFile, found, err := s.getRawFile(job.RawFileID)
	if err != nil {
		return err
	}
	if !found {
		return s.fail(jobID, "raw_file not found")
	}

	parsed, err := parsers.Parse(parsers.Input{
		ParserKind:       job.ParserKind,
		StoragePath:      rawFile.StoragePath,
		OriginalFilename: rawFile.OriginalFilename,
		RawFileSHA256:    rawFile.SHA256,
	})
	if err != nil {
		return s.fail(jobID, err.Error())
	}

	if err := s.persistActivities(rawFile, parsed); err != nil {
		return s.fail(jobID, err.Error())
	}

	finishedAt := time.Now().UTC()
	return s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":      StatusCompleted,
		"finished_at": &finishedAt,
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

func (s *Service) getRawFile(id uuid.UUID) (models.RawFile, bool, error) {
	var rawFile models.RawFile
	err := s.db.First(&rawFile, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.RawFile{}, false, nil
	}
	if err != nil {
		return models.RawFile{}, false, err
	}
	return rawFile, true, nil
}

func (s *Service) fail(jobID uuid.UUID, message string) error {
	finishedAt := time.Now().UTC()
	return s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":        StatusFailed,
		"error_message": &message,
		"finished_at":   &finishedAt,
	}).Error
}

func (s *Service) persistActivities(rawFile models.RawFile, parsed []parsers.ParsedActivity) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, activity := range parsed {
			if activity.ExternalID == "" {
				return fmt.Errorf("parsed activity missing external id")
			}
			activityID, err := s.upsertActivity(tx, rawFile, activity)
			if err != nil {
				return err
			}
			if err := replaceRoutePoints(tx, activityID, activity.RoutePoints); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) upsertActivity(tx *gorm.DB, rawFile models.RawFile, parsed parsers.ParsedActivity) (uuid.UUID, error) {
	var externalRef models.ExternalRef
	err := tx.First(&externalRef, "provider = ? and external_id = ?", parsed.Provider, parsed.ExternalID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return uuid.Nil, err
	}

	now := time.Now().UTC()
	if err == nil {
		updates := map[string]any{
			"sport_type":         parsed.SportType,
			"title":              parsed.Title,
			"started_at":         parsed.StartedAt,
			"ended_at":           parsed.EndedAt,
			"distance_m":         parsed.DistanceM,
			"duration_s":         parsed.DurationS,
			"source_kind":        parsed.SourceKind,
			"source_activity_id": parsed.SourceActivityID,
			"updated_at":         now,
		}
		return externalRef.ActivityID, tx.Model(&models.Activity{}).Where("id = ?", externalRef.ActivityID).Updates(updates).Error
	}

	activityID, err := ids.New()
	if err != nil {
		return uuid.Nil, err
	}
	refID, err := ids.New()
	if err != nil {
		return uuid.Nil, err
	}

	activity := models.Activity{
		ID:               activityID,
		SportType:        parsed.SportType,
		Title:            parsed.Title,
		StartedAt:        parsed.StartedAt,
		EndedAt:          parsed.EndedAt,
		DistanceM:        parsed.DistanceM,
		DurationS:        parsed.DurationS,
		SourceKind:       parsed.SourceKind,
		SourceActivityID: parsed.SourceActivityID,
		FirstRawFileID:   rawFile.ID,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := tx.Create(&activity).Error; err != nil {
		return uuid.Nil, err
	}

	externalRef = models.ExternalRef{
		ID:         refID,
		ActivityID: activityID,
		Provider:   parsed.Provider,
		ExternalID: parsed.ExternalID,
		RawFileID:  rawFile.ID,
		CreatedAt:  now,
	}
	if err := tx.Create(&externalRef).Error; err != nil {
		return uuid.Nil, err
	}

	return activityID, nil
}

func replaceRoutePoints(tx *gorm.DB, activityID uuid.UUID, points []parsers.RoutePoint) error {
	if err := tx.Delete(&models.ActivityRoutePoint{}, "activity_id = ?", activityID).Error; err != nil {
		return err
	}
	for seq, point := range points {
		if err := tx.Exec(
			`insert into tb_activity_route_points
			  (activity_id, seq, ts, lat, lon, elevation_m, geom)
			  values (?, ?, ?, ?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)`,
			activityID,
			seq,
			point.Ts,
			point.Lat,
			point.Lon,
			point.ElevationM,
			point.Lon,
			point.Lat,
		).Error; err != nil {
			return err
		}
	}
	return nil
}
