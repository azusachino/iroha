package imports

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
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
