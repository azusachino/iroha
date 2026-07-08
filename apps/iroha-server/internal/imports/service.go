package imports

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
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

	reused, err := s.reuseCompletedImport(jobID, job.ParserVersion, rawFile.SHA256)
	if err != nil {
		return err
	}
	if reused {
		return nil
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

	snapshotID, err := ids.New()
	if err != nil {
		return s.fail(jobID, err.Error())
	}
	snapshot := models.ImportSnapshot{
		ID:            snapshotID,
		ImportJobID:   jobID,
		RawFileID:     rawFile.ID,
		SHA256:        rawFile.SHA256,
		ParserVersion: job.ParserVersion,
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.persistActivities(rawFile, parsed, snapshot); err != nil {
		return s.fail(jobID, err.Error())
	}

	finishedAt := time.Now().UTC()
	return s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":      StatusCompleted,
		"finished_at": &finishedAt,
	}).Error
}

// reuseCompletedImport checks whether a COMPLETED import job already exists
// for a raw file with the same sha256 processed by the same parser
// version. If so, the current job is a redundant re-run against
// already-imported content (the raw file itself is deduped at upload by
// sha256, so this guards re-running the same logical import rather than
// re-uploads) - short-circuit it as completed without re-parsing or
// re-persisting anything.
func (s *Service) reuseCompletedImport(jobID uuid.UUID, parserVersion string, sha256 string) (bool, error) {
	var existing models.ImportJob
	err := s.db.
		Joins("join tb_raw_files on tb_raw_files.id = tb_import_jobs.raw_file_id").
		Where("tb_import_jobs.status = ? and tb_import_jobs.parser_version = ? and tb_raw_files.sha256 = ? and tb_import_jobs.id <> ?",
			StatusCompleted, parserVersion, sha256, jobID).
		Order("tb_import_jobs.created_at desc").
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	s.logger.Info("reusing prior completed import; skipping re-parse",
		"job_id", jobID.String(),
		"reused_job_id", existing.ID.String(),
		"sha256", sha256,
		"parser_version", parserVersion,
	)

	finishedAt := time.Now().UTC()
	if err := s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":      StatusCompleted,
		"finished_at": &finishedAt,
	}).Error; err != nil {
		return false, err
	}
	return true, nil
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

func (s *Service) persistActivities(rawFile models.RawFile, parsed []parsers.ParsedActivity, snapshot models.ImportSnapshot) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}

		for _, activity := range parsed {
			if activity.ExternalID == "" {
				return fmt.Errorf("parsed activity missing external id")
			}

			// Activities without a content hash (currently: everything but
			// Apple Health workouts) keep the original always-upsert
			// behavior; they don't participate in apple_source_items
			// change-detection.
			if activity.ContentHash == "" {
				activityID, err := s.upsertActivity(tx, rawFile, activity)
				if err != nil {
					return err
				}
				if err := replaceRoutePoints(tx, activityID, activity.RoutePoints); err != nil {
					return err
				}
				continue
			}

			if err := s.persistAppleWorkout(tx, rawFile, activity, snapshot.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// persistAppleWorkout applies per-workout change detection for a parsed
// Apple Health workout activity (identified by having a non-empty
// ContentHash): unchanged workouts skip both the activity upsert and the
// route point rewrite entirely, only bumping the source item's
// last_seen_snapshot_id so we know it was still present in this export.
func (s *Service) persistAppleWorkout(tx *gorm.DB, rawFile models.RawFile, activity parsers.ParsedActivity, snapshotID uuid.UUID) error {
	var existing models.AppleSourceItem
	res := tx.Limit(1).Find(&existing, "source_key = ?", activity.ExternalID)
	if res.Error != nil {
		return res.Error
	}
	found := res.RowsAffected > 0

	var existingHash *string
	if found {
		existingHash = &existing.ContentHash
	}

	now := time.Now().UTC()

	switch decideSourceItem(existingHash, activity.ContentHash) {
	case sourceItemUnchanged:
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error

	case sourceItemChanged:
		activityID, err := s.upsertActivity(tx, rawFile, activity)
		if err != nil {
			return err
		}
		if err := replaceRoutePoints(tx, activityID, activity.RoutePoints); err != nil {
			return err
		}
		if err := replaceLaps(tx, activityID, activity.Laps); err != nil {
			return err
		}
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"content_hash":          activity.ContentHash,
			"activity_id":           activityID,
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error

	default: // sourceItemNew
		activityID, err := s.upsertActivity(tx, rawFile, activity)
		if err != nil {
			return err
		}
		if err := replaceRoutePoints(tx, activityID, activity.RoutePoints); err != nil {
			return err
		}
		if err := replaceLaps(tx, activityID, activity.Laps); err != nil {
			return err
		}
		itemID, err := ids.New()
		if err != nil {
			return err
		}
		item := models.AppleSourceItem{
			ID:                 itemID,
			SourceKey:          activity.ExternalID,
			ItemType:           "workout",
			ContentHash:        activity.ContentHash,
			ActivityID:         &activityID,
			LastSeenSnapshotID: &snapshotID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		return tx.Create(&item).Error
	}
}

func (s *Service) upsertActivity(tx *gorm.DB, rawFile models.RawFile, parsed parsers.ParsedActivity) (uuid.UUID, error) {
	var externalRef models.ExternalRef
	res := tx.Limit(1).Find(&externalRef, "provider = ? and external_id = ?", parsed.Provider, parsed.ExternalID)
	if res.Error != nil {
		return uuid.Nil, res.Error
	}
	found := res.RowsAffected > 0

	now := time.Now().UTC()
	if found {
		updates := map[string]any{
			"sport_type":         parsed.SportType,
			"title":              parsed.Title,
			"started_at":         parsed.StartedAt,
			"ended_at":           parsed.EndedAt,
			"distance_m":         parsed.DistanceM,
			"duration_s":         parsed.DurationS,
			"avg_hr":             parsed.AvgHR,
			"max_hr":             parsed.MaxHR,
			"avg_pace_s_per_km":  parsed.AvgPaceSPerKM,
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
		AvgHR:            parsed.AvgHR,
		MaxHR:            parsed.MaxHR,
		AvgPaceSPerKM:    parsed.AvgPaceSPerKM,
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

// replaceLaps deletes any existing laps for the activity and inserts the
// newly parsed ones. Mirrors replaceRoutePoints, but laps are few per
// workout so a simple create-slice loop (rather than a hand-built batched
// multi-row INSERT) is fine here.
func replaceLaps(tx *gorm.DB, activityID uuid.UUID, laps []parsers.ParsedLap) error {
	if err := tx.Delete(&models.ActivityLap{}, "activity_id = ?", activityID).Error; err != nil {
		return err
	}
	if len(laps) == 0 {
		return nil
	}

	rows := make([]models.ActivityLap, 0, len(laps))
	for _, lap := range laps {
		id, err := ids.New()
		if err != nil {
			return err
		}
		rows = append(rows, models.ActivityLap{
			ID:            id,
			ActivityID:    activityID,
			LapNo:         lap.LapNo,
			StartTs:       lap.StartTs,
			EndTs:         lap.EndTs,
			DistanceM:     nil,
			DurationS:     lap.DurationS,
			AvgHR:         nil,
			AvgPaceSPerKM: nil,
		})
	}
	return tx.Create(&rows).Error
}

// routePointInsertBatchSize caps how many route points are inserted per
// multi-row INSERT statement. Each row binds 8 params, so 1000 rows/batch
// binds 8000 params - well under PostgreSQL's 65535 bind-parameter limit.
const routePointInsertBatchSize = 1000

func replaceRoutePoints(tx *gorm.DB, activityID uuid.UUID, points []parsers.RoutePoint) error {
	if err := tx.Delete(&models.ActivityRoutePoint{}, "activity_id = ?", activityID).Error; err != nil {
		return err
	}
	for start := 0; start < len(points); start += routePointInsertBatchSize {
		end := start + routePointInsertBatchSize
		if end > len(points) {
			end = len(points)
		}
		sql, args := buildRoutePointsInsertSQL(activityID, points[start:end], start)
		if err := tx.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

// buildRoutePointsInsertSQL builds a single multi-row INSERT statement (and
// its flat arg list) for one batch of route points. startSeq is the
// absolute index of points[0] within the full slice passed to
// replaceRoutePoints, so seq numbering stays correct across chunks.
func buildRoutePointsInsertSQL(activityID uuid.UUID, points []parsers.RoutePoint, startSeq int) (string, []any) {
	var sb strings.Builder
	sb.WriteString(`insert into tb_activity_route_points
		  (activity_id, seq, ts, lat, lon, elevation_m, geom)
		  values `)

	args := make([]any, 0, len(points)*8)
	for i, point := range points {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString("(?, ?, ?, ?, ?, ?, ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography)")
		args = append(args,
			activityID,
			startSeq+i,
			point.Ts,
			point.Lat,
			point.Lon,
			point.ElevationM,
			point.Lon,
			point.Lat,
		)
	}
	return sb.String(), args
}
