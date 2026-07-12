package imports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/cache"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/jobs"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/parsers"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusQueued    = "queued"
	StatusParsing   = "parsing"
	StatusCompleted = "completed"
	StatusFailed    = "failed"

	appleSourceItemTypeWorkout      = "workout"
	appleSourceItemTypeSleepSession = "sleep_session"
	appleSourceItemTypeDailySummary = "daily_summary"
	appleSourceItemTypeDailyMetric  = "daily_metric"
)

// DefaultParserVersion identifies the current parser build. A completed
// import at a different version triggers a reprocess (purge + re-persist)
// rather than a duplicate append; bump this when parser semantics change.
const DefaultParserVersion = "apple-health-2026-07-body-vitals"

type Enqueuer interface {
	EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error)
}

type Service struct {
	db            *gorm.DB
	logger        *slog.Logger
	parserVersion string
	enqueuer      Enqueuer
	cacheClient   *cache.Client
}

type CreateInput struct {
	RawFileID  string
	ParserKind string
}

func NewService(db *gorm.DB, logger *slog.Logger, parserVersion string, enqueuer Enqueuer, cacheClient *cache.Client) *Service {
	if parserVersion == "" {
		parserVersion = DefaultParserVersion
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, logger: logger, parserVersion: parserVersion, enqueuer: enqueuer, cacheClient: cacheClient}
}

func (s *Service) Create(input CreateInput) (models.ImportJob, error) {
	rawFileID, err := ids.Decode(ids.RawFilePrefix, input.RawFileID)
	if err != nil {
		return models.ImportJob{}, err
	}

	if err := s.ensureRawFileExists(rawFileID); err != nil {
		return models.ImportJob{}, err
	}

	var jobKind string
	switch input.ParserKind {
	case parsers.KindAppleHealthExport:
		jobKind = jobs.KindAppleImportParse
	case parsers.KindGPX:
		jobKind = jobs.KindGPXImportParse
	default:
		return models.ImportJob{}, fmt.Errorf("unsupported parser kind: %s", input.ParserKind)
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

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&job).Error; err != nil {
			return err
		}

		payload := map[string]any{
			"import_job_id": job.ID.String(),
		}
		if _, err := s.enqueuer.EnqueueTx(tx, jobKind, payload); err != nil {
			s.logger.Error("enqueue import job", "job_id", job.ID.String(), "error", err)
			return err
		}
		return nil
	})
	if err != nil {
		return models.ImportJob{}, err
	}

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
	if err := s.Process(jobID); err != nil {
		s.logger.Error("process import job", "job_id", jobID.String(), "error", err)
	}
}

func (s *Service) Process(jobID uuid.UUID) error {
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

	prior, priorFound, err := s.priorCompletedImport(jobID, rawFile.SHA256)
	if err != nil {
		return err
	}
	priorSameVersion := priorFound && prior.ParserVersion == job.ParserVersion

	switch decideImportDisposition(priorSameVersion, priorFound) {
	case dispositionSkip:
		return s.reuseCompletedImport(jobID, prior)
	case dispositionReprocess:
		s.logger.Info(
			"reprocessing import: parser_version differs from prior completed import; purging and re-persisting",
			"job_id", jobID.String(),
			"prior_job_id", prior.ID.String(),
			"prior_parser_version", prior.ParserVersion,
			"parser_version", job.ParserVersion,
			"sha256", rawFile.SHA256,
		)
	}
	reprocess := priorFound && !priorSameVersion

	parsed, err := parsers.Parse(parsers.Input{
		ParserKind:       job.ParserKind,
		StoragePath:      rawFile.StoragePath,
		OriginalFilename: rawFile.OriginalFilename,
		RawFileSHA256:    rawFile.SHA256,
	})
	if err != nil {
		return s.fail(jobID, err.Error())
	}
	var parsedSleep []parsers.ParsedSleepSession
	var parsedDailySummaries []parsers.ParsedDailySummary
	var parsedDailyMetrics []parsers.ParsedDailyMetric
	if job.ParserKind == parsers.KindAppleHealthExport {
		parsedSleep, err = parsers.ParseAppleHealthSleep(rawFile.StoragePath)
		if err != nil {
			return s.fail(jobID, err.Error())
		}
		parsedDailySummaries, parsedDailyMetrics, err = parsers.ParseAppleHealthDailyActivity(rawFile.StoragePath)
		if err != nil {
			return s.fail(jobID, err.Error())
		}
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

	if err := s.persistActivities(rawFile, parsed, parsedSleep, parsedDailySummaries, parsedDailyMetrics, snapshot, reprocess); err != nil {
		return s.fail(jobID, err.Error())
	}

	finishedAt := time.Now().UTC()
	err = s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":      StatusCompleted,
		"finished_at": &finishedAt,
	}).Error
	if err == nil {
		s.flushCache()
	}
	return err
}

// priorCompletedImport looks up the most recent COMPLETED import job for a
// raw file with the given sha256, excluding the current job (the raw file
// itself is deduped at upload by sha256, so this identifies prior completed
// imports of the same logical source rather than re-uploads). found is
// false if no such job exists.
func (s *Service) priorCompletedImport(jobID uuid.UUID, sha256 string) (models.ImportJob, bool, error) {
	var existing models.ImportJob
	err := s.db.
		Joins("join tb_raw_files on tb_raw_files.id = tb_import_jobs.raw_file_id").
		Where("tb_import_jobs.status = ? and tb_raw_files.sha256 = ? and tb_import_jobs.id <> ?",
			StatusCompleted, sha256, jobID).
		Order("tb_import_jobs.created_at desc").
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.ImportJob{}, false, nil
	}
	if err != nil {
		return models.ImportJob{}, false, err
	}
	return existing, true, nil
}

// reuseCompletedImport marks jobID as completed without re-parsing or
// re-persisting anything, because a prior completed import already covers
// the same raw file sha256 at the same parser_version (dispositionSkip).
func (s *Service) reuseCompletedImport(jobID uuid.UUID, existing models.ImportJob) error {
	s.logger.Info(
		"reusing prior completed import; skipping re-parse",
		"job_id", jobID.String(),
		"reused_job_id", existing.ID.String(),
		"parser_version", existing.ParserVersion,
	)

	finishedAt := time.Now().UTC()
	err := s.db.Model(&models.ImportJob{}).Where("id = ?", jobID).Updates(map[string]any{
		"status":      StatusCompleted,
		"finished_at": &finishedAt,
	}).Error
	if err == nil {
		s.flushCache()
	}
	return err
}

func (s *Service) flushCache() {
	if s.cacheClient == nil {
		return
	}
	s.logger.Info("flushing public cache keys after import job completion")
	ctx := context.Background()
	if err := s.cacheClient.DeletePattern(ctx, "public:*"); err != nil {
		s.logger.Error("failed to flush public cache keys", "error", err)
	}
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

// purgeDerivedForRawFile deletes everything derived from a raw file so it
// can be re-persisted fresh (dispositionReprocess), instead of appending
// alongside stale rows. The delete order matters:
//
//  1. tb_apple_source_items first. These carry the content_hash used by
//     persistAppleWorkout's change-detection. If they survived a purge, the
//     freshly re-parsed workouts would look "unchanged" against the
//     just-deleted activities and persistAppleWorkout would only bump
//     last_seen_snapshot_id instead of re-creating the activity -
//     resurrecting nothing and silently losing data. They're matched two
//     ways: by activity_id (workouts produced from this raw file) and by
//     last_seen_snapshot_id (source items last touched by a snapshot of
//     this raw file, covering unchanged workouts that were never
//     re-persisted but still got their last_seen bumped).
//  2. tb_import_snapshots for this raw file, now safe to remove since no
//     source item still references them.
//  3. tb_activities with first_raw_file_id = this raw file. ON DELETE
//     CASCADE removes tb_external_refs, tb_activity_route_points,
//     tb_activity_samplings, and tb_activity_laps for those activities.
//     This step must come after (1) since tb_apple_source_items.activity_id
//     is ON DELETE SET NULL, not CASCADE - deleting activities first would
//     just null out the source items rather than removing them, defeating
//     step (1)'s purpose.
func purgeDerivedForRawFile(tx *gorm.DB, rawFileID uuid.UUID) error {
	if err := tx.Exec(`
		delete from tb_apple_source_items
		where activity_id in (select id from tb_activities where first_raw_file_id = ?)
		   or sleep_session_id in (select id from tb_sleep_sessions where first_raw_file_id = ?)
		   or daily_summary_id in (select id from tb_daily_summaries where first_raw_file_id = ?)
		   or daily_metric_id in (select id from tb_daily_metrics where first_raw_file_id = ?)
		   or last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?)
	`, rawFileID, rawFileID, rawFileID, rawFileID, rawFileID).Error; err != nil {
		return err
	}

	if err := tx.Where("raw_file_id = ?", rawFileID).Delete(&models.ImportSnapshot{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.Activity{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.SleepSession{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.DailyMetric{}).Error; err != nil {
		return err
	}

	if err := tx.Where("first_raw_file_id = ?", rawFileID).Delete(&models.DailySummary{}).Error; err != nil {
		return err
	}

	return nil
}

func (s *Service) persistActivities(rawFile models.RawFile, parsed []parsers.ParsedActivity, parsedSleep []parsers.ParsedSleepSession, parsedDailySummaries []parsers.ParsedDailySummary, parsedDailyMetrics []parsers.ParsedDailyMetric, snapshot models.ImportSnapshot, reprocess bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if reprocess {
			if err := purgeDerivedForRawFile(tx, rawFile.ID); err != nil {
				return err
			}
		}

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
		for _, session := range parsedSleep {
			if err := s.persistSleepSession(tx, rawFile, session, snapshot.ID); err != nil {
				return err
			}
		}
		for _, summary := range parsedDailySummaries {
			if err := s.persistDailySummary(tx, rawFile, summary, snapshot.ID); err != nil {
				return err
			}
		}
		for _, metric := range parsedDailyMetrics {
			if err := s.persistDailyMetric(tx, rawFile, metric, snapshot.ID); err != nil {
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
		if err := replaceSamplings(tx, activityID, activity.Samplings); err != nil {
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
		if err := replaceSamplings(tx, activityID, activity.Samplings); err != nil {
			return err
		}
		itemID, err := ids.New()
		if err != nil {
			return err
		}
		item := models.AppleSourceItem{
			ID:                 itemID,
			SourceKey:          activity.ExternalID,
			ItemType:           appleSourceItemTypeWorkout,
			ContentHash:        activity.ContentHash,
			ActivityID:         &activityID,
			LastSeenSnapshotID: &snapshotID,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		return tx.Create(&item).Error
	}
}

func sleepSessionSourceKey(session parsers.ParsedSleepSession) string {
	return strings.Join([]string{
		session.Source,
		session.WakeDate.Format("2006-01-02"),
		session.StartedAt.Format(time.RFC3339Nano),
		session.EndedAt.Format(time.RFC3339Nano),
	}, "|")
}

func sleepSessionContentHash(session parsers.ParsedSleepSession) string {
	var content strings.Builder
	for _, segment := range session.Segments {
		fmt.Fprintf(
			&content, "%s|%s|%s|%s\n",
			segment.Stage,
			segment.StartedAt.Format(time.RFC3339Nano),
			segment.EndedAt.Format(time.RFC3339Nano),
			segment.Source,
		)
	}
	sum := sha256.Sum256([]byte(content.String()))
	return hex.EncodeToString(sum[:])
}

func (s *Service) persistSleepSession(tx *gorm.DB, rawFile models.RawFile, session parsers.ParsedSleepSession, snapshotID uuid.UUID) error {
	sourceKey := sleepSessionSourceKey(session)
	if sourceKey == "" {
		return fmt.Errorf("parsed sleep session missing source key")
	}
	contentHash := sleepSessionContentHash(session)

	var existing models.AppleSourceItem
	res := tx.Limit(1).Find(&existing, "source_key = ?", sourceKey)
	if res.Error != nil {
		return res.Error
	}
	found := res.RowsAffected > 0
	if found && existing.ItemType != appleSourceItemTypeSleepSession {
		return fmt.Errorf("source key %q already belongs to item type %q", sourceKey, existing.ItemType)
	}

	var existingHash *string
	if found {
		existingHash = &existing.ContentHash
	}
	now := time.Now().UTC()
	switch decideSourceItem(existingHash, contentHash) {
	case sourceItemUnchanged:
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}

	sessionID := uuid.Nil
	if found && existing.SleepSessionID != nil {
		sessionID = *existing.SleepSessionID
	}
	if sessionID == uuid.Nil {
		var err error
		sessionID, err = ids.New()
		if err != nil {
			return err
		}
	}
	if err := upsertSleepSession(tx, rawFile, sessionID, session); err != nil {
		return err
	}
	if err := replaceSleepSegments(tx, sessionID, session.Segments); err != nil {
		return err
	}

	if found {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"content_hash":          contentHash,
			"sleep_session_id":      sessionID,
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}
	itemID, err := ids.New()
	if err != nil {
		return err
	}
	item := models.AppleSourceItem{
		ID:                 itemID,
		SourceKey:          sourceKey,
		ItemType:           appleSourceItemTypeSleepSession,
		ContentHash:        contentHash,
		SleepSessionID:     &sessionID,
		LastSeenSnapshotID: &snapshotID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	return tx.Create(&item).Error
}

func upsertSleepSession(tx *gorm.DB, rawFile models.RawFile, sessionID uuid.UUID, parsed parsers.ParsedSleepSession) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"wake_date":     parsed.WakeDate,
		"started_at":    parsed.StartedAt,
		"ended_at":      parsed.EndedAt,
		"time_in_bed_s": parsed.TimeInBedS,
		"asleep_s":      parsed.AsleepS,
		"efficiency":    parsed.Efficiency,
		"is_main_sleep": parsed.IsMainSleep,
		"core_s":        parsed.CoreS,
		"deep_s":        parsed.DeepS,
		"rem_s":         parsed.RemS,
		"awake_s":       parsed.AwakeS,
		"unspecified_s": parsed.UnspecifiedS,
		"source":        parsed.Source,
		"updated_at":    now,
	}
	var existing models.SleepSession
	if err := tx.First(&existing, "id = ?", sessionID).Error; err == nil {
		return tx.Model(&models.SleepSession{}).Where("id = ?", sessionID).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return tx.Create(&models.SleepSession{
		ID:             sessionID,
		WakeDate:       parsed.WakeDate,
		StartedAt:      parsed.StartedAt,
		EndedAt:        parsed.EndedAt,
		TimeInBedS:     parsed.TimeInBedS,
		AsleepS:        parsed.AsleepS,
		Efficiency:     parsed.Efficiency,
		IsMainSleep:    parsed.IsMainSleep,
		CoreS:          parsed.CoreS,
		DeepS:          parsed.DeepS,
		RemS:           parsed.RemS,
		AwakeS:         parsed.AwakeS,
		UnspecifiedS:   parsed.UnspecifiedS,
		Source:         parsed.Source,
		FirstRawFileID: rawFile.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error
}

const sleepSegmentInsertBatchSize = 1000

func replaceSleepSegments(tx *gorm.DB, sessionID uuid.UUID, segments []parsers.ParsedSleepSegment) error {
	if err := tx.Delete(&models.SleepSegment{}, "session_id = ?", sessionID).Error; err != nil {
		return err
	}
	if len(segments) == 0 {
		return nil
	}
	rows := make([]models.SleepSegment, 0, len(segments))
	for seq, segment := range segments {
		id, err := ids.New()
		if err != nil {
			return err
		}
		rows = append(rows, models.SleepSegment{
			ID:        id,
			SessionID: sessionID,
			Stage:     segment.Stage,
			StartedAt: segment.StartedAt,
			EndedAt:   segment.EndedAt,
			Seq:       seq,
		})
	}
	return tx.CreateInBatches(rows, sleepSegmentInsertBatchSize).Error
}

func dailySummarySourceKey(summary parsers.ParsedDailySummary) string {
	return summary.Day.Format("2006-01-02")
}

func dailySummaryContentHash(summary parsers.ParsedDailySummary) string {
	content := fmt.Sprintf(
		"%s|%.17g|%.17g|%.17g|%.17g|%.17g|%.17g|%s",
		dailySummarySourceKey(summary),
		summary.MoveKcal,
		summary.MoveGoalKcal,
		summary.ExerciseMin,
		summary.ExerciseGoalMin,
		summary.StandHours,
		summary.StandGoalHours,
		summary.Source,
	)
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func dailyMetricSourceKey(metric parsers.ParsedDailyMetric) string {
	return strings.Join([]string{metric.Day.Format("2006-01-02"), metric.Metric}, "|")
}

func dailyMetricContentHash(metric parsers.ParsedDailyMetric) string {
	content := fmt.Sprintf("%s|%.17g|%s|%s", dailyMetricSourceKey(metric), metric.Value, metric.Unit, metric.Source)
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func (s *Service) persistDailySummary(tx *gorm.DB, rawFile models.RawFile, summary parsers.ParsedDailySummary, snapshotID uuid.UUID) error {
	sourceKey := dailySummarySourceKey(summary)
	if sourceKey == "" {
		return fmt.Errorf("parsed daily summary missing source key")
	}
	contentHash := dailySummaryContentHash(summary)

	var existing models.AppleSourceItem
	res := tx.Limit(1).Find(&existing, "source_key = ?", sourceKey)
	if res.Error != nil {
		return res.Error
	}
	found := res.RowsAffected > 0
	if found && existing.ItemType != appleSourceItemTypeDailySummary {
		return fmt.Errorf("source key %q already belongs to item type %q", sourceKey, existing.ItemType)
	}
	if found && existing.ContentHash == contentHash {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            time.Now().UTC(),
		}).Error
	}

	summaryID := uuid.Nil
	if found && existing.DailySummaryID != nil {
		summaryID = *existing.DailySummaryID
	}
	if summaryID == uuid.Nil {
		var err error
		summaryID, err = ids.New()
		if err != nil {
			return err
		}
	}
	if err := upsertDailySummary(tx, rawFile, summaryID, summary); err != nil {
		return err
	}
	now := time.Now().UTC()
	if found {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"content_hash":          contentHash,
			"daily_summary_id":      summaryID,
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}
	itemID, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Create(&models.AppleSourceItem{
		ID:                 itemID,
		SourceKey:          sourceKey,
		ItemType:           appleSourceItemTypeDailySummary,
		ContentHash:        contentHash,
		DailySummaryID:     &summaryID,
		LastSeenSnapshotID: &snapshotID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error
}

func upsertDailySummary(tx *gorm.DB, rawFile models.RawFile, summaryID uuid.UUID, parsed parsers.ParsedDailySummary) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"day":               parsed.Day,
		"move_kcal":         parsed.MoveKcal,
		"move_goal_kcal":    parsed.MoveGoalKcal,
		"exercise_min":      parsed.ExerciseMin,
		"exercise_goal_min": parsed.ExerciseGoalMin,
		"stand_hours":       parsed.StandHours,
		"stand_goal_hours":  parsed.StandGoalHours,
		"source":            parsed.Source,
		"updated_at":        now,
	}
	var existing models.DailySummary
	if err := tx.First(&existing, "id = ?", summaryID).Error; err == nil {
		return tx.Model(&models.DailySummary{}).Where("id = ?", summaryID).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Create(&models.DailySummary{
		ID:              summaryID,
		Day:             parsed.Day,
		MoveKcal:        parsed.MoveKcal,
		MoveGoalKcal:    parsed.MoveGoalKcal,
		ExerciseMin:     parsed.ExerciseMin,
		ExerciseGoalMin: parsed.ExerciseGoalMin,
		StandHours:      parsed.StandHours,
		StandGoalHours:  parsed.StandGoalHours,
		Source:          parsed.Source,
		FirstRawFileID:  rawFile.ID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error
}

func (s *Service) persistDailyMetric(tx *gorm.DB, rawFile models.RawFile, metric parsers.ParsedDailyMetric, snapshotID uuid.UUID) error {
	sourceKey := dailyMetricSourceKey(metric)
	if sourceKey == "" {
		return fmt.Errorf("parsed daily metric missing source key")
	}
	contentHash := dailyMetricContentHash(metric)

	var existing models.AppleSourceItem
	res := tx.Limit(1).Find(&existing, "source_key = ?", sourceKey)
	if res.Error != nil {
		return res.Error
	}
	found := res.RowsAffected > 0
	if found && existing.ItemType != appleSourceItemTypeDailyMetric {
		return fmt.Errorf("source key %q already belongs to item type %q", sourceKey, existing.ItemType)
	}
	if found && existing.ContentHash == contentHash {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            time.Now().UTC(),
		}).Error
	}

	metricID := uuid.Nil
	if found && existing.DailyMetricID != nil {
		metricID = *existing.DailyMetricID
	}
	if metricID == uuid.Nil {
		var err error
		metricID, err = ids.New()
		if err != nil {
			return err
		}
	}
	if err := upsertDailyMetric(tx, rawFile, metricID, metric); err != nil {
		return err
	}
	now := time.Now().UTC()
	if found {
		return tx.Model(&models.AppleSourceItem{}).Where("id = ?", existing.ID).Updates(map[string]any{
			"content_hash":          contentHash,
			"daily_metric_id":       metricID,
			"last_seen_snapshot_id": snapshotID,
			"updated_at":            now,
		}).Error
	}
	itemID, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Create(&models.AppleSourceItem{
		ID:                 itemID,
		SourceKey:          sourceKey,
		ItemType:           appleSourceItemTypeDailyMetric,
		ContentHash:        contentHash,
		DailyMetricID:      &metricID,
		LastSeenSnapshotID: &snapshotID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}).Error
}

func upsertDailyMetric(tx *gorm.DB, rawFile models.RawFile, metricID uuid.UUID, parsed parsers.ParsedDailyMetric) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"day":        parsed.Day,
		"metric":     parsed.Metric,
		"value":      parsed.Value,
		"unit":       parsed.Unit,
		"source":     parsed.Source,
		"updated_at": now,
	}
	var existing models.DailyMetric
	if err := tx.First(&existing, "id = ?", metricID).Error; err == nil {
		return tx.Model(&models.DailyMetric{}).Where("id = ?", metricID).Updates(updates).Error
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return tx.Create(&models.DailyMetric{
		ID:             metricID,
		Day:            parsed.Day,
		Metric:         parsed.Metric,
		Value:          parsed.Value,
		Unit:           parsed.Unit,
		Source:         parsed.Source,
		FirstRawFileID: rawFile.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error
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
		CaloriesKcal:     parsed.CaloriesKcal,
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
			CaloriesKcal:  lap.CaloriesKcal,
		})
	}
	return tx.Create(&rows).Error
}

// samplingInsertBatchSize caps how many samplings are inserted per batch via
// GORM's CreateInBatches. Unlike route points, samplings have no PostGIS
// geom column to hand-build SQL for, so CreateInBatches is clean here.
const samplingInsertBatchSize = 1000

// replaceSamplings deletes any existing samplings for the activity and
// bulk-inserts the newly parsed ones. Deliberately NOT called for the
// sourceItemUnchanged branch of persistAppleWorkout (an unchanged workout
// has unchanged samples, and re-writing thousands of samples per unchanged
// workout would be wasted work over ~2M source records) and not called on
// the non-Apple import path.
func replaceSamplings(tx *gorm.DB, activityID uuid.UUID, samplings []parsers.ParsedSampling) error {
	if err := tx.Delete(&models.ActivitySampling{}, "activity_id = ?", activityID).Error; err != nil {
		return err
	}
	if len(samplings) == 0 {
		return nil
	}

	rows := make([]models.ActivitySampling, 0, len(samplings))
	for _, sampling := range samplings {
		id, err := ids.New()
		if err != nil {
			return err
		}
		rows = append(rows, models.ActivitySampling{
			ID:           id,
			ActivityID:   activityID,
			SamplingType: sampling.SamplingType,
			Ts:           sampling.Ts,
			Value:        sampling.Value,
			Unit:         sampling.Unit,
		})
	}
	return tx.CreateInBatches(rows, samplingInsertBatchSize).Error
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
		args = append(
			args,
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
