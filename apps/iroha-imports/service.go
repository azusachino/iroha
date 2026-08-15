package imports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-core/observations"
	provider "github.com/azusachino/iroha/apps/iroha-core/provider/v1"
	providerregistry "github.com/azusachino/iroha/apps/iroha-providers/registry"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	mediaWorkKind     = "media"
	mediaItemRole     = "primary"
	mediaListKind     = "library"
	mediaScopeType    = "item"
	mediaUnknownValue = "unknown"
)

// DefaultParserVersion identifies the current parser build. A completed
// import at a different version triggers a reprocess (purge + re-persist)
// rather than a duplicate append; bump this when parser semantics change.
const DefaultParserVersion = coreimports.DefaultParserVersion

type Enqueuer interface {
	EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error)
}

type Service struct {
	db            *gorm.DB
	logger        *slog.Logger
	parserVersion string
	enqueuer      Enqueuer
	cacheClient   *cache.Client
	providers     *provider.Registry
	mediaBridge   MediaRefBridge
}

type CreateInput struct {
	RawFileID  string
	ParserKind string
}

func NewService(db *gorm.DB, logger *slog.Logger, parserVersion string, enqueuer Enqueuer, cacheClient *cache.Client) *Service {
	providers, err := providerregistry.New()
	if err != nil {
		panic(fmt.Sprintf("build provider registry: %v", err))
	}
	return NewServiceWithRegistry(db, logger, parserVersion, enqueuer, cacheClient, providers)
}

func NewServiceWithRegistry(db *gorm.DB, logger *slog.Logger, parserVersion string, enqueuer Enqueuer, cacheClient *cache.Client, providers *provider.Registry) *Service {
	return NewServiceWithRegistryAndBridge(db, logger, parserVersion, enqueuer, cacheClient, providers, nil)
}

func NewServiceWithRegistryAndBridge(db *gorm.DB, logger *slog.Logger, parserVersion string, enqueuer Enqueuer, cacheClient *cache.Client, providers *provider.Registry, mediaBridge MediaRefBridge) *Service {
	if parserVersion == "" {
		parserVersion = DefaultParserVersion
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, logger: logger, parserVersion: parserVersion, enqueuer: enqueuer, cacheClient: cacheClient, providers: providers, mediaBridge: mediaBridge}
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
	case coreimports.KindAppleHealthExport:
		jobKind = jobs.KindAppleImportParse
	case coreimports.KindGPX:
		jobKind = jobs.KindGPXImportParse
	case coreimports.KindAniList, coreimports.KindAniListActivity, coreimports.KindBangumi:
		jobKind = jobs.KindMediaIntakeParse
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
	if err := s.ProcessContext(context.Background(), jobID); err != nil {
		s.logger.Error("process import job", "job_id", jobID.String(), "error", err)
	}
}

func (s *Service) Process(jobID uuid.UUID) error {
	return s.ProcessContext(context.Background(), jobID)
}

func (s *Service) ProcessContext(ctx context.Context, jobID uuid.UUID) error {
	if err := ctx.Err(); err != nil {
		return err
	}
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
	disposition := decideImportDisposition(priorSameVersion, priorFound)

	switch disposition {
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
	reprocess := disposition == dispositionReprocess

	adapter, ok := s.providers.GetBySourceKind(job.ParserKind)
	if !ok {
		return s.fail(jobID, fmt.Sprintf("no provider adapter for source kind %q", job.ParserKind))
	}
	source := provider.Source{
		Kind:             job.ParserKind,
		OriginalFilename: rawFile.OriginalFilename,
		SHA256:           rawFile.SHA256,
		Open: func(context.Context) (io.ReadCloser, error) {
			return os.Open(rawFile.StoragePath)
		},
	}
	options := provider.ImportOptions{}
	var parsedMedia []observations.Media
	var parsedMediaHistory []observations.MediaHistory
	var parsed []observations.Activity
	var parsedSleep []observations.Sleep
	var parsedDailySummaries []observations.DailySummary
	var parsedDailyMetrics []observations.DailyMetric
	mediaImporter, mediaOK := adapter.(provider.MediaImporter)
	mediaHistoryImporter, mediaHistoryOK := adapter.(provider.MediaHistoryImporter)
	if job.ParserKind == coreimports.KindAniListActivity {
		if !mediaHistoryOK {
			return s.fail(jobID, fmt.Sprintf("provider %q does not implement media history import", adapter.Descriptor().ID))
		}
		parsedMediaHistory, err = mediaHistoryImporter.ImportMediaHistory(ctx, source, options)
		if err != nil {
			return s.fail(jobID, err.Error())
		}
	} else if mediaOK {
		parsedMedia, err = mediaImporter.ImportMedia(ctx, source, options)
		if err != nil {
			return s.fail(jobID, err.Error())
		}
	} else if batchImporter, ok := adapter.(provider.BatchImporter); ok {
		batch, batchErr := batchImporter.ImportAll(ctx, source, options)
		if batchErr != nil {
			return s.fail(jobID, batchErr.Error())
		}
		parsed = batch.Activities
		parsedSleep = batch.Sleep
		parsedDailySummaries = batch.Daily.Summaries
		parsedDailyMetrics = batch.Daily.Metrics
	} else {
		activityImporter, activityOK := adapter.(provider.ActivityImporter)
		if !activityOK {
			return s.fail(jobID, fmt.Sprintf("provider %q does not implement activity import", adapter.Descriptor().ID))
		}
		parsed, err = activityImporter.ImportActivities(ctx, source, options)
		if err != nil {
			return s.fail(jobID, err.Error())
		}
		if sleepImporter, sleepOK := adapter.(provider.SleepImporter); sleepOK {
			parsedSleep, err = sleepImporter.ImportSleep(ctx, source, options)
			if err != nil {
				return s.fail(jobID, err.Error())
			}
		}
		if dailyImporter, dailyOK := adapter.(provider.DailyImporter); dailyOK {
			daily, dailyErr := dailyImporter.ImportDaily(ctx, source, options)
			if dailyErr != nil {
				return s.fail(jobID, dailyErr.Error())
			}
			parsedDailySummaries = daily.Summaries
			parsedDailyMetrics = daily.Metrics
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
	if rawFile.ObservedAt != nil {
		snapshot.TakenAt = rawFile.ObservedAt
	} else {
		// Manual uploads have no source-observation clock. Their ingestion
		// time is retained only as the Iroha-observed basis.
		snapshot.TakenAt = &rawFile.CreatedAt
	}

	if job.ParserKind == coreimports.KindAniListActivity {
		err = s.persistMediaHistory(rawFile, parsedMediaHistory, snapshot, reprocess)
	} else if mediaOK {
		err = s.persistMedia(rawFile, parsedMedia, snapshot, reprocess)
	} else {
		err = s.persistActivities(rawFile, parsed, parsedSleep, parsedDailySummaries, parsedDailyMetrics, snapshot, reprocess)
	}
	if err != nil {
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
	s.logger.Info("invalidating read cache namespaces after import job completion", "change", cache.ChangeImport)
	if err := s.cacheClient.InvalidateChange(context.Background(), cache.ChangeImport); err != nil {
		s.logger.Error("failed to invalidate caches after import", "error", err)
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
//     source item or source observation still references them.
//  3. tb_activities with first_raw_file_id = this raw file. ON DELETE
//     CASCADE removes tb_external_refs, tb_activity_route_points,
//     tb_activity_samplings, and tb_activity_laps for those activities.
//     This step must come after (1) since tb_apple_source_items.activity_id
//     is ON DELETE SET NULL, not CASCADE - deleting activities first would
//     just null out the source items rather than removing them, defeating
//     step (1)'s purpose.
func purgeDerivedForRawFile(tx *gorm.DB, rawFileID uuid.UUID) error {
	if err := tx.Where("raw_file_id = ?", rawFileID).Delete(&models.MediaConsumptionEvent{}).Error; err != nil {
		return err
	}
	if err := tx.Where("raw_file_id = ?", rawFileID).Delete(&models.MediaStateHistory{}).Error; err != nil {
		return err
	}

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
	for _, statement := range []string{
		`update tb_activities set selected_observation_id = null where selected_observation_id in (select ao.id from tb_activity_observations ao join tb_source_observations so on so.id = ao.id where so.raw_file_id = ? or so.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or so.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
		`update tb_sleep_sessions set selected_observation_id = null where selected_observation_id in (select so.id from tb_sleep_observations so join tb_source_observations src on src.id = so.id where src.raw_file_id = ? or src.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or src.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
		`update tb_daily_summaries set selected_observation_id = null where selected_observation_id in (select so.id from tb_daily_summary_observations so join tb_source_observations src on src.id = so.id where src.raw_file_id = ? or src.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or src.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
		`update tb_daily_metrics set selected_observation_id = null where selected_observation_id in (select so.id from tb_daily_metric_observations so join tb_source_observations src on src.id = so.id where src.raw_file_id = ? or src.first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?) or src.last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?))`,
	} {
		if err := tx.Exec(statement, rawFileID, rawFileID, rawFileID).Error; err != nil {
			return err
		}
	}
	if err := tx.Exec(`
		delete from tb_source_observations
		where raw_file_id = ?
		   or first_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?)
		   or last_seen_snapshot_id in (select id from tb_import_snapshots where raw_file_id = ?)
	`, rawFileID, rawFileID, rawFileID).Error; err != nil {
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

func (s *Service) persistMedia(rawFile models.RawFile, parsed []observations.Media, snapshot models.ImportSnapshot, reprocess bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if reprocess {
			if err := purgeDerivedForRawFile(tx, rawFile.ID); err != nil {
				return err
			}
		}

		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		for _, media := range parsed {
			if err := persistMediaObservation(tx, rawFile, snapshot, media, s.mediaBridge); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Service) persistMediaHistory(rawFile models.RawFile, parsed []observations.MediaHistory, snapshot models.ImportSnapshot, reprocess bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if reprocess {
			if err := purgeDerivedForRawFile(tx, rawFile.ID); err != nil {
				return err
			}
		}
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		for _, history := range parsed {
			itemID, err := ensureMediaItem(tx, history.Media, s.mediaBridge)
			if err != nil {
				return err
			}
			if err := persistMediaMetadata(tx, itemID, history.Media); err != nil {
				return err
			}
			if err := persistMediaRelations(tx, itemID, history.Media); err != nil {
				return err
			}
			for _, update := range history.Updates {
				if err := persistMediaStateUpdate(tx, rawFile, snapshot, itemID, update); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func persistMediaStateUpdate(tx *gorm.DB, rawFile models.RawFile, snapshot models.ImportSnapshot, itemID uuid.UUID, update observations.MediaStateUpdate) error {
	if update.SourceEventID == "" {
		return errors.New("media provider activity missing source event id")
	}
	if update.EffectiveAt.IsZero() {
		return fmt.Errorf("media provider activity %q is missing effective_at", update.SourceEventID)
	}
	update.EffectiveAt = update.EffectiveAt.UTC()
	fingerprintBytes, err := json.Marshal(update)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(fingerprintBytes)
	fingerprint := hex.EncodeToString(digest[:])
	const sourceKind = coreimports.KindAniList
	var existing models.MediaStateHistory
	result := tx.Where("media_item_id = ? and source_kind = ? and source_event_id = ?", itemID, sourceKind, update.SourceEventID).
		Order("observed_at desc, created_at desc, id desc").First(&existing)
	if result.Error == nil && existing.StateFingerprint == fingerprint {
		return nil
	}
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	observedAt := rawFile.CreatedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	if rawFile.ObservedAt != nil {
		observedAt = *rawFile.ObservedAt
	}
	if snapshot.TakenAt != nil && !snapshot.TakenAt.IsZero() {
		observedAt = *snapshot.TakenAt
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.MediaStateHistory{
		ID:               id,
		MediaItemID:      itemID,
		SourceKind:       sourceKind,
		SourceEventID:    update.SourceEventID,
		ObservedAt:       observedAt.UTC(),
		EffectiveAt:      &update.EffectiveAt,
		TimeBasis:        "provider_activity",
		ChangeKind:       "provider_activity",
		StateFingerprint: fingerprint,
		Status:           update.Status,
		Unit:             update.Unit,
		Position:         update.Position,
		Total:            update.Total,
		ProgressPercent:  update.ProgressPercent,
		Rating:           update.Rating,
		RatingScale:      update.RatingScale,
		Note:             update.Note,
		RepeatCount:      update.RepeatCount,
		RawFileID:        &rawFile.ID,
		CreatedAt:        time.Now().UTC(),
	}).Error
}

func persistMediaObservation(tx *gorm.DB, rawFile models.RawFile, snapshot models.ImportSnapshot, media observations.Media, bridge MediaRefBridge) error {
	itemID, err := ensureMediaItem(tx, media, bridge)
	if err != nil {
		return err
	}

	if err := persistMediaMetadata(tx, itemID, media); err != nil {
		return err
	}
	listName := media.Provider + " library"
	var list models.MediaList
	result := tx.Where("source_kind = ? and name = ?", media.Provider, listName).First(&list)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		listID, err := ids.New()
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		list = models.MediaList{
			ID:         listID,
			Name:       listName,
			ListKind:   mediaListKind,
			SourceKind: media.Provider,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&list).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	}
	var listItem models.MediaListItem
	result = tx.Where("list_id = ? and media_item_id = ?", list.ID, itemID).First(&listItem)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		listItemID, err := ids.New()
		if err != nil {
			return err
		}
		if err := tx.Create(&models.MediaListItem{ID: listItemID, ListID: list.ID, MediaItemID: itemID, CreatedAt: time.Now().UTC()}).Error; err != nil {
			return err
		}
	} else if result.Error != nil {
		return result.Error
	}

	if err := persistMediaRelations(tx, itemID, media); err != nil {
		return err
	}
	if err := persistMediaEvents(tx, rawFile, itemID, media); err != nil {
		return err
	}
	if err := persistMediaStateHistory(tx, rawFile, snapshot, itemID, media); err != nil {
		return err
	}
	return upsertMediaProgress(tx, rawFile, itemID, media)
}

func ensureMediaItem(tx *gorm.DB, media observations.Media, bridge MediaRefBridge) (uuid.UUID, error) {
	if media.Provider == "" {
		return uuid.Nil, errors.New("parsed media missing provider")
	}
	if media.ExternalID == "" {
		return uuid.Nil, errors.New("parsed media missing external id")
	}
	if media.Title == "" {
		return uuid.Nil, errors.New("parsed media missing title")
	}

	resolution, err := resolveMediaItem(tx, media, bridge)
	if err != nil {
		return uuid.Nil, err
	}
	var externalRef models.MediaExternalRef
	if resolution.ItemID == uuid.Nil {
		workID, err := ids.New()
		if err != nil {
			return uuid.Nil, err
		}
		itemID, err := ids.New()
		if err != nil {
			return uuid.Nil, err
		}
		now := time.Now().UTC()
		work := models.MediaWork{
			ID:            workID,
			WorkKind:      mediaWorkKind,
			PrimaryTitle:  media.Title,
			OriginalTitle: media.Title,
			Description:   media.Description,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := tx.Create(&work).Error; err != nil {
			return uuid.Nil, err
		}
		releaseDate, releasePrecision := canonicalReleaseDate(media)
		item := models.MediaItem{
			ID:                   itemID,
			WorkID:               &workID,
			MediaType:            media.MediaType,
			ItemRole:             itemRoleOrDefault(media.ItemRole),
			Title:                media.Title,
			OriginalTitle:        media.Title,
			ReleaseDate:          releaseDate,
			ReleaseDatePrecision: releasePrecision,
			SeasonNumber:         media.SeasonNumber,
			EpisodeNumber:        media.EpisodeNumber,
			ChapterNumber:        media.ChapterNumber,
			VolumeNumber:         media.VolumeNumber,
			DurationSeconds:      media.DurationSeconds,
			PageCount:            media.PageCount,
			EpisodeCount:         media.EpisodeCount,
			ChapterCount:         media.ChapterCount,
			Language:             media.Language,
			Country:              media.Country,
			CoverImageURL:        media.CoverImageURL,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		if item.MediaType == "" {
			item.MediaType = mediaUnknownValue
		}
		if err := tx.Create(&item).Error; err != nil {
			return uuid.Nil, err
		}
		externalRefID, err := ids.New()
		if err != nil {
			return uuid.Nil, err
		}
		externalRef = models.MediaExternalRef{
			ID:         externalRefID,
			ScopeType:  mediaScopeType,
			ScopeID:    itemID,
			Provider:   media.Provider,
			ExternalID: media.ExternalID,
			MatchedBy:  "provider_id",
			CreatedAt:  now,
		}
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "provider"}, {Name: "external_id"}},
			DoNothing: true,
		}).Create(&externalRef)
		if result.Error != nil {
			return uuid.Nil, result.Error
		}
		if result.RowsAffected == 0 {
			var existing models.MediaExternalRef
			if err := tx.Where("provider = ? and external_id = ?", media.Provider, media.ExternalID).First(&existing).Error; err != nil {
				return uuid.Nil, err
			}
			if err := tx.Delete(&models.MediaItem{}, itemID).Error; err != nil {
				return uuid.Nil, err
			}
			if err := tx.Delete(&models.MediaWork{}, workID).Error; err != nil {
				return uuid.Nil, err
			}
			resolution.ItemID = existing.ScopeID
		}
		if resolution.ItemID == uuid.Nil {
			resolution.ItemID = itemID
		}
	} else {
		if err := requireMediaResolution(resolution); err != nil {
			return uuid.Nil, err
		}
		itemID := resolution.ItemID
		// The item is "owned" by this provider when its own primary ref already
		// pointed here before this sync: then a fresh parse (e.g. a reprocess
		// after a parser fix) may overwrite the item's core fields. If the item
		// was reached via a bridge/title match from a different provider, only
		// fill empty fields so we don't clobber the owner's values each sync.
		ownedItem := true
		lookupErr := tx.Where("scope_type = ? and scope_id = ? and provider = ? and external_id = ?", mediaScopeType, itemID, media.Provider, media.ExternalID).First(&externalRef).Error
		if errors.Is(lookupErr, gorm.ErrRecordNotFound) {
			ownedItem = false
			refID, idErr := ids.New()
			if idErr != nil {
				return uuid.Nil, idErr
			}
			// INSERT ... ON CONFLICT DO NOTHING: (provider, external_id) is
			// unique across all items, so a concurrent job may have already
			// claimed this ref for a different item while we were resolving.
			newRef := models.MediaExternalRef{ID: refID, ScopeType: mediaScopeType, ScopeID: itemID, Provider: media.Provider, ExternalID: media.ExternalID, MatchedBy: resolution.MatchedBy, Confidence: resolution.Confidence, CreatedAt: time.Now().UTC()}
			result := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "provider"}, {Name: "external_id"}},
				DoNothing: true,
			}).Create(&newRef)
			if result.Error != nil {
				return uuid.Nil, result.Error
			}
			if result.RowsAffected == 0 {
				var existing models.MediaExternalRef
				if err := tx.Where("provider = ? and external_id = ?", media.Provider, media.ExternalID).First(&existing).Error; err != nil {
					return uuid.Nil, err
				}
				if existing.ScopeID != itemID {
					return uuid.Nil, createExternalRefConflictTask(tx, itemID, observations.MediaExternalRef{Provider: media.Provider, ExternalID: media.ExternalID}, existing.ScopeID)
				}
				externalRef = existing
			} else {
				externalRef = newRef
			}
		} else if lookupErr != nil {
			return uuid.Nil, lookupErr
		}
		if err := refreshMediaItemFields(tx, itemID, media, ownedItem); err != nil {
			return uuid.Nil, err
		}
	}
	return resolution.ItemID, nil
}

// persistMediaRelations writes the provider relation graph (adaptation/sequel/
// etc.) into tb_media_relations. Both endpoints must already resolve to items
// via external refs; edges to items not (yet) in the collection are skipped
// rather than materialized as stub items. The unique constraint dedupes edges
// re-observed across syncs.
func persistMediaRelations(tx *gorm.DB, itemID uuid.UUID, media observations.Media) error {
	for _, rel := range media.Relations {
		if rel.RelationType == "" || rel.ToExternalID == "" {
			continue
		}
		toRef, err := findExternalRef(tx, rel.Provider, rel.ToExternalID)
		if err != nil {
			return err
		}
		if toRef == nil {
			continue
		}
		fromID := itemID
		if rel.FromExternalID != "" && rel.FromExternalID != media.ExternalID {
			fromRef, err := findExternalRef(tx, rel.Provider, rel.FromExternalID)
			if err != nil {
				return err
			}
			if fromRef == nil {
				continue
			}
			fromID = fromRef.ScopeID
		}
		id, err := ids.New()
		if err != nil {
			return err
		}
		relation := models.MediaRelation{
			ID: id, FromType: mediaScopeType, FromID: fromID, ToType: mediaScopeType, ToID: toRef.ScopeID,
			RelationType: rel.RelationType, Provider: rel.Provider, Confidence: rel.Confidence, CreatedAt: time.Now().UTC(),
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relation).Error; err != nil {
			return err
		}
	}
	return nil
}

// persistMediaEvents appends only exact events explicitly supplied by an
// adapter. Provider list snapshots do not enter this table, and there is no
// fallback from a fuzzy completion date or import time.
func persistMediaEvents(tx *gorm.DB, rawFile models.RawFile, itemID uuid.UUID, media observations.Media) error {
	for _, event := range media.Events {
		if !observations.ValidMediaEventType(event.EventType) {
			return fmt.Errorf("media exact event has invalid event type %q", event.EventType)
		}
		if event.EventAt.IsZero() {
			return fmt.Errorf("media exact event %q is missing event_at", event.EventType)
		}
		event.EventAt = event.EventAt.UTC()
		unchanged, err := latestEventUnchanged(tx, itemID, rawFile.SourceKind, event)
		if err != nil {
			return err
		}
		if unchanged {
			continue
		}
		id, err := ids.New()
		if err != nil {
			return err
		}
		if err := tx.Create(&models.MediaConsumptionEvent{
			ID:              id,
			MediaItemID:     itemID,
			EventType:       event.EventType,
			EventAt:         event.EventAt,
			SourceKind:      rawFile.SourceKind,
			SourceEventID:   event.SourceEventID,
			Unit:            event.Unit,
			Position:        event.Position,
			Total:           event.Total,
			ProgressPercent: event.ProgressPercent,
			Rating:          event.Rating,
			RatingScale:     event.RatingScale,
			Note:            event.Note,
			RawFileID:       &rawFile.ID,
			CreatedAt:       time.Now().UTC(),
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

// persistMediaStateHistory records the provider's typed current state. The
// fingerprint is calculated from canonical fields, so repeated snapshots and
// reprocessing do not create a second observation. A provider record timestamp
// remains ordering/provenance only; observed_at is when Iroha saw the state.
func persistMediaStateHistory(tx *gorm.DB, rawFile models.RawFile, snapshot models.ImportSnapshot, itemID uuid.UUID, media observations.Media) error {
	observedAt := rawFile.CreatedAt
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	if rawFile.ObservedAt != nil {
		observedAt = *rawFile.ObservedAt
	}
	if snapshot.TakenAt != nil && !snapshot.TakenAt.IsZero() {
		observedAt = *snapshot.TakenAt
	}
	observedAt = observedAt.UTC()
	sourceEventID := media.StateSourceID
	if sourceEventID == "" {
		sourceEventID = media.ExternalID
	}
	startedValue, startedPrecision := partialDateFields(media.StartedOn)
	completedValue, completedPrecision := partialDateFields(media.CompletedOn)
	state := struct {
		Status               string
		Unit                 string
		Position             *float64
		Total                *float64
		ProgressPercent      *float64
		Rating               *float64
		RatingScale          *float64
		Note                 string
		RepeatCount          int
		StartedOnValue       *time.Time
		StartedOnPrecision   string
		CompletedOnValue     *time.Time
		CompletedOnPrecision string
	}{
		Status: media.Status, Position: media.Progress, Rating: media.Score, Note: media.StateNote,
		StartedOnValue: startedValue, StartedOnPrecision: startedPrecision,
		CompletedOnValue: completedValue, CompletedOnPrecision: completedPrecision,
	}
	if media.ProgressState != nil {
		state.Status = media.ProgressState.Status
		state.Unit = media.ProgressState.Unit
		state.Position = media.ProgressState.Position
		state.Total = media.ProgressState.Total
		state.ProgressPercent = media.ProgressState.ProgressPercent
		state.RepeatCount = media.ProgressState.PlayCount
		if state.StartedOnValue == nil {
			state.StartedOnValue, state.StartedOnPrecision = partialDateFields(media.ProgressState.StartedOn)
		}
		if state.CompletedOnValue == nil {
			state.CompletedOnValue, state.CompletedOnPrecision = partialDateFields(media.ProgressState.CompletedOn)
		}
	}
	state.RatingScale = media.StateRatingScale
	effectiveOnValue, effectiveOnPrecision := (*time.Time)(nil), ""
	if state.CompletedOnPrecision == string(observations.DatePrecisionDay) {
		effectiveOnValue, effectiveOnPrecision = state.CompletedOnValue, state.CompletedOnPrecision
	}
	fingerprintBytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(fingerprintBytes)
	fingerprint := hex.EncodeToString(digest[:])

	var existing models.MediaStateHistory
	result := tx.Where("media_item_id = ? and source_kind = ? and source_event_id = ?", itemID, rawFile.SourceKind, sourceEventID).
		Order("observed_at desc, created_at desc, id desc").First(&existing)
	if result.Error == nil && existing.StateFingerprint == fingerprint {
		return nil
	}
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return result.Error
	}
	changeKind := "snapshot"
	if result.Error == nil {
		changeKind = "changed"
	}
	// Provider timestamps describe when the upstream record changed, but a
	// current list snapshot is not an activity event. Keep that timestamp as
	// provenance only; the state observation itself is still Iroha-observed
	// until a connector supplies a real provider activity record.
	timeBasis := "iroha_observed"
	var providerRecordedAt *time.Time
	if media.ProgressState != nil && media.ProgressState.LastUpdateAt != nil {
		providerRecordedAt = media.ProgressState.LastUpdateAt
	}
	if effectiveOnPrecision == string(observations.DatePrecisionDay) {
		timeBasis = "source_date"
	}
	id, err := ids.New()
	if err != nil {
		return err
	}
	result = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&models.MediaStateHistory{
		ID: id, MediaItemID: itemID, SourceKind: rawFile.SourceKind, SourceEventID: sourceEventID,
		ObservedAt: observedAt, TimeBasis: timeBasis, ChangeKind: changeKind, StateFingerprint: fingerprint,
		Status: state.Status, Unit: state.Unit, Position: state.Position, Total: state.Total,
		ProgressPercent: state.ProgressPercent, Rating: state.Rating, RatingScale: state.RatingScale,
		Note: state.Note, RepeatCount: state.RepeatCount, StartedOnValue: state.StartedOnValue,
		StartedOnPrecision: state.StartedOnPrecision, CompletedOnValue: state.CompletedOnValue,
		CompletedOnPrecision: state.CompletedOnPrecision, EffectiveOnValue: effectiveOnValue,
		EffectiveOnPrecision: effectiveOnPrecision, ProviderRecordedAt: providerRecordedAt,
		RawFileID: &rawFile.ID, CreatedAt: time.Now().UTC(),
	})
	if result.Error != nil {
		return result.Error
	}
	// The raw-file-scoped fingerprint index makes concurrent retries of the
	// same snapshot idempotent without preventing a legitimate A -> B -> A
	// state transition across later observations.
	return nil
}

func partialDateFields(value *observations.PartialDate) (*time.Time, string) {
	if value == nil || !value.Valid() {
		return nil, ""
	}
	date := value.Value
	return &date, string(value.Precision)
}

// latestEventUnchanged reports whether the most recent event for the same
// source entry already records identical progress/rating/note, meaning a fresh
// sync observed no change. Events without a stable source id always append.
func latestEventUnchanged(tx *gorm.DB, itemID uuid.UUID, sourceKind string, event observations.MediaEvent) (bool, error) {
	if event.SourceEventID == "" {
		return false, nil
	}
	var existing models.MediaConsumptionEvent
	err := tx.Where("media_item_id = ? and source_kind = ? and source_event_id = ? and event_type = ?",
		itemID, sourceKind, event.SourceEventID, event.EventType).
		Order("event_at desc, created_at desc").
		First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return existing.EventAt.Equal(event.EventAt) &&
		existing.Unit == event.Unit &&
		floatPtrEqual(existing.Position, event.Position) &&
		floatPtrEqual(existing.Total, event.Total) &&
		floatPtrEqual(existing.Rating, event.Rating) &&
		floatPtrEqual(existing.ProgressPercent, event.ProgressPercent) &&
		floatPtrEqual(existing.RatingScale, event.RatingScale) &&
		existing.Note == event.Note, nil
}

// upsertMediaProgress recomputes the current-progress projection, preferring
// the adapter's rich ProgressState (unit/play_count/last_update/hidden) and
// falling back to flat fields. A cross-source status disagreement is routed to
// the inbox instead of silently overwriting.
func upsertMediaProgress(tx *gorm.DB, rawFile models.RawFile, itemID uuid.UUID, media observations.Media) error {
	progress := models.MediaProgress{
		MediaItemID: itemID,
		Status:      media.Status,
		Position:    media.Progress,
		SourceKind:  rawFile.SourceKind,
		UpdatedAt:   time.Now().UTC(),
	}
	progress.StartedOnValue, progress.StartedOnPrecision = partialDateFields(media.StartedOn)
	progress.CompletedOnValue, progress.CompletedOnPrecision = partialDateFields(media.CompletedOn)
	if ps := media.ProgressState; ps != nil {
		progress.Status = ps.Status
		progress.Unit = ps.Unit
		progress.Position = ps.Position
		progress.Total = ps.Total
		progress.ProgressPercent = ps.ProgressPercent
		progress.StartedOnValue, progress.StartedOnPrecision = partialDateFields(ps.StartedOn)
		progress.LastUpdateAt = ps.LastUpdateAt
		progress.CompletedOnValue, progress.CompletedOnPrecision = partialDateFields(ps.CompletedOn)
		progress.PlayCount = ps.PlayCount
		// Paused/on-hold entries stay status=in_progress but must not surface in
		// the "continue" strip; fold the adapter's Paused flag into the column.
		progress.HiddenFromContinue = ps.HiddenFromContinue || ps.Paused
	}
	if progress.Status == "" {
		progress.Status = mediaUnknownValue
	}
	var existing models.MediaProgress
	result := tx.Where("media_item_id = ?", itemID).First(&existing)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return tx.Create(&progress).Error
	}
	if result.Error != nil {
		return result.Error
	}
	if existing.SourceKind != "" && existing.SourceKind != rawFile.SourceKind && existing.Status != progress.Status {
		return createProgressConflictTask(tx, media, itemID, existing.Status, progress.Status)
	}
	return tx.Model(&existing).Updates(map[string]any{
		"status":                 progress.Status,
		"unit":                   progress.Unit,
		"position":               progress.Position,
		"total":                  progress.Total,
		"progress_percent":       progress.ProgressPercent,
		"started_on_value":       progress.StartedOnValue,
		"started_on_precision":   progress.StartedOnPrecision,
		"last_update_at":         progress.LastUpdateAt,
		"completed_on_value":     progress.CompletedOnValue,
		"completed_on_precision": progress.CompletedOnPrecision,
		"play_count":             progress.PlayCount,
		"hidden_from_continue":   progress.HiddenFromContinue,
		"source_kind":            progress.SourceKind,
		"updated_at":             progress.UpdatedAt,
	}).Error
}

func floatPtrEqual(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func itemRoleOrDefault(role string) string {
	if role == "" {
		return mediaItemRole
	}
	return role
}

func canonicalReleaseDate(media observations.Media) (*time.Time, string) {
	if media.ReleaseDateOn != nil && media.ReleaseDateOn.Valid() {
		date := media.ReleaseDateOn.Value
		return &date, string(media.ReleaseDateOn.Precision)
	}
	if media.ReleaseDate != nil {
		return media.ReleaseDate, string(observations.DatePrecisionDay)
	}
	return nil, ""
}

// titleLanguageRank scores a title string for the JPN > ENG > CHN display
// precedence: kana (hiragana/katakana) marks it distinctly Japanese even
// mixed with kanji, since kana never appears in Chinese; CJK ideographs
// with no kana are treated as Chinese (Bangumi's name_cn); anything else
// (Latin script -- English/romaji) ranks in between.
func titleLanguageRank(title string) int {
	const (
		rankJapanese = 1
		rankEnglish  = 2
		rankChinese  = 3
	)
	hasKana := false
	hasCJK := false
	for _, r := range title {
		switch {
		case r >= 0x3040 && r <= 0x30FF:
			hasKana = true
		case r >= 0x4E00 && r <= 0x9FFF:
			hasCJK = true
		}
	}
	switch {
	case hasKana:
		return rankJapanese
	case hasCJK:
		return rankChinese
	default:
		return rankEnglish
	}
}

// refreshMediaItemFields updates the core columns of an already-existing item
// from a fresh observation. When owned (the item's own provider is syncing) a
// non-empty incoming value overwrites, so a reprocess after a parser fix
// re-applies; otherwise only empty columns are filled, so a bridge/title match
// from another provider never clobbers the owner's data.
func refreshMediaItemFields(tx *gorm.DB, itemID uuid.UUID, media observations.Media, owned bool) error {
	var item models.MediaItem
	if err := tx.First(&item, "id = ?", itemID).Error; err != nil {
		return err
	}
	updates := map[string]any{}
	setStr := func(col, existing, incoming string) {
		if incoming == "" || incoming == existing {
			return
		}
		if owned || existing == "" {
			updates[col] = incoming
		}
	}
	setInt := func(col string, existing, incoming *int) {
		if incoming == nil {
			return
		}
		if owned || existing == nil {
			updates[col] = *incoming
		}
	}
	setFloat := func(col string, existing, incoming *float64) {
		if incoming == nil {
			return
		}
		if owned || existing == nil {
			updates[col] = *incoming
		}
	}

	// Title precedence is by script, not the generic owned/existing-empty rule
	// above, and not by provider: Bangumi's own subject.Name (used whenever
	// subject.NameCN is empty) is Japanese too, so "prefer AniList" would be
	// the wrong proxy for "prefer Japanese." Rank JPN > ENG > CHN and only
	// replace the existing title with a strictly higher-ranked incoming one.
	if media.Title != "" && media.Title != item.Title {
		if item.Title == "" || titleLanguageRank(media.Title) < titleLanguageRank(item.Title) {
			updates["title"] = media.Title
			updates["original_title"] = media.Title
		}
	}

	if media.MediaType != "" && media.MediaType != mediaUnknownValue {
		setStr("media_type", item.MediaType, media.MediaType)
	}
	if media.ItemRole != "" {
		setStr("item_role", item.ItemRole, media.ItemRole)
	}
	if releaseDate, precision := canonicalReleaseDate(media); releaseDate != nil && (owned || item.ReleaseDate == nil) {
		updates["release_date"] = releaseDate
		updates["release_date_precision"] = precision
	}
	setInt("season_number", item.SeasonNumber, media.SeasonNumber)
	setInt("episode_number", item.EpisodeNumber, media.EpisodeNumber)
	setFloat("chapter_number", item.ChapterNumber, media.ChapterNumber)
	setFloat("volume_number", item.VolumeNumber, media.VolumeNumber)
	setInt("duration_seconds", item.DurationSeconds, media.DurationSeconds)
	setInt("page_count", item.PageCount, media.PageCount)
	setInt("episode_count", item.EpisodeCount, media.EpisodeCount)
	setInt("chapter_count", item.ChapterCount, media.ChapterCount)
	setStr("language", item.Language, media.Language)
	setStr("country", item.Country, media.Country)
	setStr("cover_image_url", item.CoverImageURL, media.CoverImageURL)

	if len(updates) > 0 {
		updates["updated_at"] = time.Now().UTC()
		if err := tx.Model(&models.MediaItem{}).Where("id = ?", itemID).Updates(updates).Error; err != nil {
			return err
		}
	}

	// Description and title live on the work, not the item -- description
	// follows the same owned/existing-empty rule as the item fields above;
	// title follows the same JPN > ENG > CHN language-rank rule as item.title,
	// since tb_media_works.primary_title/original_title were seeded from the
	// same media.Title value at creation and would otherwise drift from it.
	if item.WorkID != nil && (media.Description != "" || media.Title != "") {
		var work models.MediaWork
		if err := tx.First(&work, "id = ?", *item.WorkID).Error; err != nil {
			return err
		}
		workUpdates := map[string]any{}
		if media.Description != "" && (owned || work.Description == "") && work.Description != media.Description {
			workUpdates["description"] = media.Description
		}
		if media.Title != "" && media.Title != work.PrimaryTitle &&
			(work.PrimaryTitle == "" || titleLanguageRank(media.Title) < titleLanguageRank(work.PrimaryTitle)) {
			workUpdates["primary_title"] = media.Title
			workUpdates["original_title"] = media.Title
		}
		if len(workUpdates) > 0 {
			workUpdates["updated_at"] = time.Now().UTC()
			if err := tx.Model(&models.MediaWork{}).Where("id = ?", *item.WorkID).Updates(workUpdates).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) persistActivities(rawFile models.RawFile, parsed []observations.Activity, parsedSleep []observations.Sleep, parsedDailySummaries []observations.DailySummary, parsedDailyMetrics []observations.DailyMetric, snapshot models.ImportSnapshot, reprocess bool) error {
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
				if err := s.persistActivityObservation(tx, rawFile, activity, activityID, snapshot.ID); err != nil {
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
		if rawFile.SourceKind == coreimports.KindAppleHealthExport {
			if err := reconcileCompleteAppleSnapshot(tx, snapshot.ID); err != nil {
				return err
			}
		}
		return nil
	})
}

// reconcileCompleteAppleSnapshot removes source items that disappeared from
// the latest complete Apple export. Raw files and import snapshots remain as
// immutable evidence; only the current canonical projection is reconciled.
// Source items are deleted before their derived rows because their foreign
// keys use ON DELETE SET NULL and would otherwise erase the IDs needed for
// cleanup.
func reconcileCompleteAppleSnapshot(tx *gorm.DB, snapshotID uuid.UUID) error {
	var stale []models.AppleSourceItem
	if err := tx.Where("last_seen_snapshot_id is distinct from ?", snapshotID).Find(&stale).Error; err != nil {
		return err
	}
	if len(stale) == 0 {
		return nil
	}

	activityIDs := make([]uuid.UUID, 0, len(stale))
	sleepIDs := make([]uuid.UUID, 0, len(stale))
	dailySummaryIDs := make([]uuid.UUID, 0, len(stale))
	dailyMetricIDs := make([]uuid.UUID, 0, len(stale))
	itemIDs := make([]uuid.UUID, 0, len(stale))
	for _, item := range stale {
		itemIDs = append(itemIDs, item.ID)
		if item.ActivityID != nil {
			activityIDs = append(activityIDs, *item.ActivityID)
		}
		if item.SleepSessionID != nil {
			sleepIDs = append(sleepIDs, *item.SleepSessionID)
		}
		if item.DailySummaryID != nil {
			dailySummaryIDs = append(dailySummaryIDs, *item.DailySummaryID)
		}
		if item.DailyMetricID != nil {
			dailyMetricIDs = append(dailyMetricIDs, *item.DailyMetricID)
		}
	}

	if err := tx.Where("id IN ?", itemIDs).Delete(&models.AppleSourceItem{}).Error; err != nil {
		return err
	}
	if len(activityIDs) > 0 {
		if err := tx.Where("id IN ?", activityIDs).Delete(&models.Activity{}).Error; err != nil {
			return err
		}
	}
	if len(sleepIDs) > 0 {
		if err := tx.Where("id IN ?", sleepIDs).Delete(&models.SleepSession{}).Error; err != nil {
			return err
		}
	}
	if len(dailySummaryIDs) > 0 {
		if err := tx.Where("id IN ?", dailySummaryIDs).Delete(&models.DailySummary{}).Error; err != nil {
			return err
		}
	}
	if len(dailyMetricIDs) > 0 {
		if err := tx.Where("id IN ?", dailyMetricIDs).Delete(&models.DailyMetric{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// persistAppleWorkout applies per-workout change detection for a parsed
// Apple Health workout activity (identified by having a non-empty
// ContentHash): unchanged workouts skip both the activity upsert and the
// route point rewrite entirely, only bumping the source item's
// last_seen_snapshot_id so we know it was still present in this export.
func (s *Service) persistAppleWorkout(tx *gorm.DB, rawFile models.RawFile, activity observations.Activity, snapshotID uuid.UUID) error {
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
		if err := s.persistActivityObservation(tx, rawFile, activity, activityID, snapshotID); err != nil {
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
		if err := s.persistActivityObservation(tx, rawFile, activity, activityID, snapshotID); err != nil {
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

func (s *Service) persistActivityObservation(tx *gorm.DB, rawFile models.RawFile, activity observations.Activity, activityID, snapshotID uuid.UUID) error {
	observationID, err := upsertSourceObservation(tx, rawFile, "activity", activity.Provider, activity.ExternalID, activity.ContentHash, snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.ActivityObservation{
		ID:               observationID,
		ActivityID:       activityID,
		SourceActivityID: activity.SourceActivityID,
		SportType:        activity.SportType,
		Title:            activity.Title,
		StartedAt:        activity.StartedAt,
		EndedAt:          activity.EndedAt,
		DistanceM:        activity.DistanceM,
		DurationS:        activity.DurationS,
		AvgHR:            activity.AvgHR,
		MaxHR:            activity.MaxHR,
		AvgPaceSPerKM:    activity.AvgPaceSPerKM,
		CaloriesKcal:     activity.CaloriesKcal,
		MatchStatus:      "canonical",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	var existing models.ActivityObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.ActivityObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"activity_id": activityID, "source_activity_id": activity.SourceActivityID, "sport_type": activity.SportType,
		"title": activity.Title, "started_at": activity.StartedAt, "ended_at": activity.EndedAt,
		"distance_m": activity.DistanceM, "duration_s": activity.DurationS, "avg_hr": activity.AvgHR,
		"max_hr": activity.MaxHR, "avg_pace_s_per_km": activity.AvgPaceSPerKM, "calories_kcal": activity.CaloriesKcal,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.Activity{}).Where("id = ?", activityID).Update("selected_observation_id", observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_activity_observation_route_points where activity_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`insert into tb_activity_observation_route_points (activity_observation_id, seq, ts, lat, lon, elevation_m, distance_m, speed_mps, heart_rate, geom)
select ?, seq, ts, lat, lon, elevation_m, distance_m, speed_mps, heart_rate, geom from tb_activity_route_points where activity_id = ?`, observationID, activityID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_activity_observation_samplings where activity_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`insert into tb_activity_observation_samplings (id, activity_observation_id, sampling_type, ts, value, unit)
select id, ?, sampling_type, ts, value, unit from tb_activity_samplings where activity_id = ?`, observationID, activityID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_activity_observation_laps where activity_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	return tx.Exec(`insert into tb_activity_observation_laps (id, activity_observation_id, lap_no, start_ts, end_ts, distance_m, duration_s, avg_hr, avg_pace_s_per_km, calories_kcal)
select id, ?, lap_no, start_ts, end_ts, distance_m, duration_s, avg_hr, avg_pace_s_per_km, calories_kcal from tb_activity_laps where activity_id = ?`, observationID, activityID).Error
}

func upsertSourceObservation(tx *gorm.DB, rawFile models.RawFile, sourceKind, provider, sourceKey, contentHash string, snapshotID uuid.UUID) (uuid.UUID, error) {
	var existing models.SourceObservation
	err := tx.Where("provider = ? and source_kind = ? and source_key = ?", provider, sourceKind, sourceKey).First(&existing).Error
	now := time.Now().UTC()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		id, idErr := ids.New()
		if idErr != nil {
			return uuid.Nil, idErr
		}
		first := snapshotID
		row := models.SourceObservation{ID: id, Provider: provider, SourceKind: sourceKind, SourceKey: sourceKey, ContentHash: contentHash, RawFileID: rawFile.ID, FirstSeenSnapshotID: &first, LastSeenSnapshotID: &snapshotID, CreatedAt: now, UpdatedAt: now}
		return id, tx.Create(&row).Error
	}
	if err != nil {
		return uuid.Nil, err
	}
	return existing.ID, tx.Model(&models.SourceObservation{}).Where("id = ?", existing.ID).Updates(map[string]any{
		"content_hash": contentHash, "raw_file_id": rawFile.ID, "last_seen_snapshot_id": snapshotID, "updated_at": now,
	}).Error
}

func sleepSessionSourceKey(session observations.Sleep) string {
	return strings.Join([]string{
		session.Source,
		session.WakeDate.Format("2006-01-02"),
		session.StartedAt.Format(time.RFC3339Nano),
		session.EndedAt.Format(time.RFC3339Nano),
	}, "|")
}

func sleepSessionContentHash(session observations.Sleep) string {
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

func (s *Service) persistSleepSession(tx *gorm.DB, rawFile models.RawFile, session observations.Sleep, snapshotID uuid.UUID) error {
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
	if err := s.persistSleepObservation(tx, rawFile, session, sessionID, snapshotID); err != nil {
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

func upsertSleepSession(tx *gorm.DB, rawFile models.RawFile, sessionID uuid.UUID, parsed observations.Sleep) error {
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

func replaceSleepSegments(tx *gorm.DB, sessionID uuid.UUID, segments []observations.SleepSegment) error {
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

func (s *Service) persistSleepObservation(tx *gorm.DB, rawFile models.RawFile, session observations.Sleep, sessionID, snapshotID uuid.UUID) error {
	contentHash := sleepSessionContentHash(session)
	observationID, err := upsertSourceObservation(tx, rawFile, "sleep", "apple_health", sleepSessionSourceKey(session), contentHash, snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.SleepObservation{ID: observationID, SleepSessionID: sessionID, WakeDate: session.WakeDate, StartedAt: session.StartedAt, EndedAt: session.EndedAt, TimeInBedS: session.TimeInBedS, AsleepS: session.AsleepS, Efficiency: session.Efficiency, IsMainSleep: session.IsMainSleep, CoreS: session.CoreS, DeepS: session.DeepS, RemS: session.RemS, AwakeS: session.AwakeS, UnspecifiedS: session.UnspecifiedS, Source: session.Source, MatchStatus: "canonical", CreatedAt: now, UpdatedAt: now}
	var existing models.SleepObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.SleepObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"sleep_session_id": sessionID, "wake_date": session.WakeDate, "started_at": session.StartedAt, "ended_at": session.EndedAt,
		"time_in_bed_s": session.TimeInBedS, "asleep_s": session.AsleepS, "efficiency": session.Efficiency, "is_main_sleep": session.IsMainSleep,
		"core_s": session.CoreS, "deep_s": session.DeepS, "rem_s": session.RemS, "awake_s": session.AwakeS, "unspecified_s": session.UnspecifiedS,
		"source": session.Source, "updated_at": now,
	}).Error; err != nil {
		return err
	}
	if err := tx.Model(&models.SleepSession{}).Where("id = ?", sessionID).Update("selected_observation_id", observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`insert into tb_sleep_session_observations (sleep_session_id, sleep_observation_id, is_preferred)
values (?, ?, true) on conflict (sleep_session_id, sleep_observation_id) do update set is_preferred = excluded.is_preferred`, sessionID, observationID).Error; err != nil {
		return err
	}
	if err := tx.Exec(`delete from tb_sleep_observation_segments where sleep_observation_id = ?`, observationID).Error; err != nil {
		return err
	}
	return tx.Exec(`insert into tb_sleep_observation_segments (id, sleep_observation_id, stage, started_at, ended_at, seq)
select id, ?, stage, started_at, ended_at, seq from tb_sleep_segments where session_id = ?`, observationID, sessionID).Error
}

func dailySummarySourceKey(summary observations.DailySummary) string {
	return summary.Day.Format("2006-01-02")
}

func dailySummaryContentHash(summary observations.DailySummary) string {
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

func dailyMetricSourceKey(metric observations.DailyMetric) string {
	return strings.Join([]string{metric.Day.Format("2006-01-02"), metric.Metric}, "|")
}

func dailyMetricContentHash(metric observations.DailyMetric) string {
	content := fmt.Sprintf("%s|%.17g|%s|%s", dailyMetricSourceKey(metric), metric.Value, metric.Unit, metric.Source)
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func (s *Service) persistDailySummary(tx *gorm.DB, rawFile models.RawFile, summary observations.DailySummary, snapshotID uuid.UUID) error {
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
	if err := s.persistDailySummaryObservation(tx, rawFile, summary, summaryID, snapshotID); err != nil {
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

func upsertDailySummary(tx *gorm.DB, rawFile models.RawFile, summaryID uuid.UUID, parsed observations.DailySummary) error {
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

func (s *Service) persistDailySummaryObservation(tx *gorm.DB, rawFile models.RawFile, summary observations.DailySummary, summaryID, snapshotID uuid.UUID) error {
	observationID, err := upsertSourceObservation(tx, rawFile, "daily_summary", "apple_health", dailySummarySourceKey(summary), dailySummaryContentHash(summary), snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.DailySummaryObservation{ID: observationID, DailySummaryID: summaryID, Day: summary.Day, MoveKcal: summary.MoveKcal, MoveGoalKcal: summary.MoveGoalKcal, ExerciseMin: summary.ExerciseMin, ExerciseGoalMin: summary.ExerciseGoalMin, StandHours: summary.StandHours, StandGoalHours: summary.StandGoalHours, Source: summary.Source, MatchStatus: "canonical", CreatedAt: now, UpdatedAt: now}
	var existing models.DailySummaryObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.DailySummaryObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"daily_summary_id": summaryID, "day": summary.Day, "move_kcal": summary.MoveKcal, "move_goal_kcal": summary.MoveGoalKcal,
		"exercise_min": summary.ExerciseMin, "exercise_goal_min": summary.ExerciseGoalMin, "stand_hours": summary.StandHours,
		"stand_goal_hours": summary.StandGoalHours, "source": summary.Source, "updated_at": now,
	}).Error; err != nil {
		return err
	}
	return tx.Model(&models.DailySummary{}).Where("id = ?", summaryID).Update("selected_observation_id", observationID).Error
}

func (s *Service) persistDailyMetric(tx *gorm.DB, rawFile models.RawFile, metric observations.DailyMetric, snapshotID uuid.UUID) error {
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
	if err := s.persistDailyMetricObservation(tx, rawFile, metric, metricID, snapshotID); err != nil {
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

func upsertDailyMetric(tx *gorm.DB, rawFile models.RawFile, metricID uuid.UUID, parsed observations.DailyMetric) error {
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

func (s *Service) persistDailyMetricObservation(tx *gorm.DB, rawFile models.RawFile, metric observations.DailyMetric, metricID, snapshotID uuid.UUID) error {
	observationID, err := upsertSourceObservation(tx, rawFile, "daily_metric", "apple_health", dailyMetricSourceKey(metric), dailyMetricContentHash(metric), snapshotID)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	row := models.DailyMetricObservation{ID: observationID, DailyMetricID: metricID, Day: metric.Day, Metric: metric.Metric, Value: metric.Value, Unit: metric.Unit, Source: metric.Source, Reducer: "source_priority", MatchStatus: "canonical", CreatedAt: now, UpdatedAt: now}
	var existing models.DailyMetricObservation
	if err := tx.First(&existing, "id = ?", observationID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if err := tx.Model(&models.DailyMetricObservation{}).Where("id = ?", observationID).Updates(map[string]any{
		"daily_metric_id": metricID, "day": metric.Day, "metric": metric.Metric, "value": metric.Value,
		"unit": metric.Unit, "source": metric.Source, "updated_at": now,
	}).Error; err != nil {
		return err
	}
	return tx.Model(&models.DailyMetric{}).Where("id = ?", metricID).Update("selected_observation_id", observationID).Error
}

func (s *Service) upsertActivity(tx *gorm.DB, rawFile models.RawFile, parsed observations.Activity) (uuid.UUID, error) {
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
func replaceLaps(tx *gorm.DB, activityID uuid.UUID, laps []observations.Lap) error {
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
func replaceSamplings(tx *gorm.DB, activityID uuid.UUID, samplings []observations.Sampling) error {
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

func replaceRoutePoints(tx *gorm.DB, activityID uuid.UUID, points []observations.RoutePoint) error {
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
func buildRoutePointsInsertSQL(activityID uuid.UUID, points []observations.RoutePoint, startSeq int) (string, []any) {
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
