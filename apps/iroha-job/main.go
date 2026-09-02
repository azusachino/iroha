package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	connector "github.com/azusachino/iroha/apps/iroha-core/connector/v1"
	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-providers/anilist"
	"github.com/azusachino/iroha/apps/iroha-providers/bangumi"
	connectorregistry "github.com/azusachino/iroha/apps/iroha-providers/connectors"
	providerregistry "github.com/azusachino/iroha/apps/iroha-providers/registry"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/dbconnect"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/publicexport"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// defaultPublicExportInterval is the projection_refresh schedule's cadence.
// jobs.ScheduleKindInterval has no cron/timezone support, so this drifts
// against wall-clock time across restarts and run durations — acceptable
// for a personal data snapshot, unlike the fixed daily-at-midnight-JST
// CronJob this schedule replaces.
const defaultPublicExportInterval = "24h"

func main() {
	once := flag.Bool("once", false, "process at most one due schedule and one queued job")
	pollInterval := flag.Duration("poll-interval", time.Second, "idle backoff between polls when the queue is empty")
	concurrency := flag.Int("concurrency", 4, "number of worker goroutines draining the queue")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load("iroha.toml")
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := dbconnect.Connect(cfg.Database.URL, &gorm.Config{}, logger)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}

	workerID, err := defaultWorkerID()
	if err != nil {
		logger.Error("build worker id", "error", err)
		os.Exit(1)
	}

	parserVersion := os.Getenv("IROHA_PARSER_VERSION")
	if parserVersion == "" {
		parserVersion = coreimports.DefaultParserVersion
	}

	cacheClient, err := cache.NewBackend(cfg.Cache.Backend, cfg.Cache.URL, db)
	if err != nil {
		logger.Error("create cache", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := cacheClient.Close(); err != nil {
			logger.Warn("close cache client", "error", err)
		}
	}()

	var jobsService *jobs.Service
	enqueuer := &workerEnqueuer{service: &jobsService}
	providers, err := providerregistry.New()
	if err != nil {
		logger.Error("build provider registry", "error", err)
		os.Exit(1)
	}
	mediaBridge, err := imports.NewReloadableMediaRefBridge(db)
	if err != nil {
		logger.Error("load media bridge", "error", err)
		os.Exit(1)
	}
	importService := imports.NewServiceWithRegistryAndBridge(db, logger, parserVersion, enqueuer, cacheClient, providers, mediaBridge)
	rawFileService, err := rawfiles.NewService(db, cfg.Storage.DataDir)
	if err != nil {
		logger.Error("create raw file service", "error", err)
		os.Exit(1)
	}
	activityConnector := anilist.NewActivityConnector(os.Getenv(config.EnvAniListUsername), os.Getenv(config.EnvAniListToken))
	if value := os.Getenv(config.EnvAniListActivityLookbackDays); value != "" {
		days, parseErr := strconv.Atoi(value)
		if parseErr != nil || days <= 0 {
			logger.Warn("invalid AniList activity lookback days; using default", "value", value, "default_days", int(anilist.DefaultActivityLookback/(24*time.Hour)))
		} else {
			activityConnector.Lookback = time.Duration(days) * 24 * time.Hour
		}
	}
	connectors, err := connectorregistry.New(
		anilist.NewConnector(os.Getenv(config.EnvAniListUsername), os.Getenv(config.EnvAniListToken)),
		activityConnector,
		bangumi.NewConnector(os.Getenv(config.EnvBangumiUsername), os.Getenv(config.EnvBangumiToken)),
	)
	if err != nil {
		logger.Error("build connector registry", "error", err)
		os.Exit(1)
	}
	syncRunner := imports.NewSyncRunner(db, connectors, rawFileService, importService)

	// Both import kinds run the same pipeline (Process dispatches on the job's
	// parser_kind); only parsers.IsImplemented kinds are ever enqueued, so only
	// those get a handler. Register couples each kind to its typed payload.
	registry := jobs.NewRegistry()
	importHandler := importParseHandler(importService)
	jobs.Register(registry, jobs.KindAppleImportParse, importHandler)
	jobs.Register(registry, jobs.KindGPXImportParse, importHandler)
	jobs.Register(registry, jobs.KindMediaIntakeParse, importHandler)
	jobs.Register(registry, jobs.KindMediaSyncAniList, mediaSyncHandler(syncRunner, "anilist"))
	jobs.Register(registry, jobs.KindMediaSyncBangumi, mediaSyncHandler(syncRunner, "bangumi"))
	jobs.Register(registry, jobs.KindMediaBridgeRefresh, mediaBridgeRefreshHandler(db, mediaBridge))

	jobsService = jobs.NewService(db, logger, registry.Handlers())
	geocodeService := geocode.NewService(db, nil, cacheClient)
	jobs.Register(registry, jobs.KindGeocodeRefresh, func(ctx context.Context, payload geocode.RefreshPayload) error {
		return geocodeService.Refresh(ctx, payload)
	})

	if exportDir := os.Getenv(config.EnvPublicExportDir); exportDir != "" {
		activityService := activities.NewService(db)
		exportPrivacy := os.Getenv(config.EnvPublicExportPrivacy) == "true"
		jobs.Register(registry, jobs.KindProjectionRefresh, publicExportRefreshHandler(activityService, geocodeService, exportDir, exportPrivacy))
		if err := ensureSchedule(db, jobs.KindProjectionRefresh, jobs.ScheduleKindInterval, defaultPublicExportInterval); err != nil {
			logger.Error("ensure public export schedule", "error", err)
			os.Exit(1)
		}
	} else {
		logger.Info("IROHA_PUBLIC_EXPORT_DIR not set; projection_refresh job kind disabled")
	}

	jobsService = jobs.NewService(db, logger, registry.Handlers())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cleanupCache(ctx, cacheClient, logger)
	if !*once {
		go runCacheMaintenance(ctx, cacheClient, logger)
	}

	if *once {
		if _, err := jobsService.EnqueueDueSchedules(1); err != nil {
			logger.Error("enqueue due schedules", "error", err)
		}
		if _, err := jobsService.ProcessNext(ctx, workerID); err != nil && !errors.Is(err, jobs.ErrNoJobAvailable) {
			logger.Error("process job", "error", err)
		}
		return
	}

	logger.Info("worker pool starting", "concurrency", *concurrency, "poll_interval", pollInterval.String())
	jobsService.Run(ctx, jobs.RunConfig{
		WorkerID:     workerID,
		Concurrency:  *concurrency,
		PollInterval: *pollInterval,
	})
}

func cleanupCache(ctx context.Context, cacheClient *cache.Client, logger *slog.Logger) {
	result, err := cacheClient.Cleanup(ctx, cache.DefaultCleanupBatchSize)
	if err != nil {
		logger.Error("cleanup cache", "error", err)
		return
	}
	if result.DeletedEntries > 0 {
		logger.Info("cleaned disposable cache entries", "deleted_entries", result.DeletedEntries)
	}
}

func runCacheMaintenance(ctx context.Context, cacheClient *cache.Client, logger *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupCache(ctx, cacheClient, logger)
		}
	}
}

func defaultWorkerID() (string, error) {
	if value := os.Getenv("IROHA_WORKER_ID"); value != "" {
		return value, nil
	}
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%d", hostname, os.Getpid()), nil
}

type workerEnqueuer struct {
	service **jobs.Service
}

func (e *workerEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	if e.service == nil || *e.service == nil {
		return models.Job{}, errors.New("job service is not initialized")
	}
	return (*e.service).EnqueueTx(tx, jobs.EnqueueInput{Kind: kind, Payload: payload})
}

type importParsePayload struct {
	ImportJobID string `json:"import_job_id"`
}

func importParseHandler(importService *imports.Service) func(context.Context, importParsePayload) error {
	return func(ctx context.Context, payload importParsePayload) error {
		if payload.ImportJobID == "" {
			return fmt.Errorf("import_job_id is required in payload")
		}
		importJobID, err := uuid.Parse(payload.ImportJobID)
		if err != nil {
			return fmt.Errorf("invalid import_job_id UUID: %w", err)
		}
		return importService.ProcessContext(ctx, importJobID)
	}
}

type mediaSyncPayload struct {
	ConnectorID string                `json:"connector_id"`
	Credentials connector.Credentials `json:"credentials"`
}

func mediaSyncHandler(runner *imports.SyncRunner, connectorID string) func(context.Context, mediaSyncPayload) error {
	return func(ctx context.Context, payload mediaSyncPayload) error {
		if payload.ConnectorID == "" {
			payload.ConnectorID = connectorID
		}
		if err := runner.Run(ctx, payload.ConnectorID, payload.Credentials); err != nil {
			return err
		}
		if payload.ConnectorID == "anilist" {
			return runner.Run(ctx, anilist.ActivityConnectorID, payload.Credentials)
		}
		return nil
	}
}

// mediaBridgeRefreshHandler re-fetches the Bangumi->MAL->AniList crosswalk
// and reloads this worker's in-memory bridge immediately afterward, so a
// manual trigger (the /to-go inbox's "Refresh media bridge" action) takes
// effect on the very next media sync, not on some later poll or restart.
func mediaBridgeRefreshHandler(db *gorm.DB, bridge *imports.ReloadableMediaRefBridge) func(context.Context, struct{}) error {
	return func(ctx context.Context, _ struct{}) error {
		if err := imports.RefreshMediaRefBridge(ctx, db); err != nil {
			return err
		}
		return bridge.Reload(db)
	}
}

// publicExportRefreshHandler regenerates the sanitized public-site snapshot
// in-process, replacing the standalone iroha-export-public CLI's scheduled
// use. publicexport.Export validates before writing and swaps outDir into
// place atomically, so any failure here — a validation error, a write
// error — surfaces as a real handler error for jobs.Service.Fail's
// retry/attempts to act on, rather than leaving a partial export in place.
func publicExportRefreshHandler(activityService *activities.Service, geocodeService *geocode.Service, outDir string, privacy bool) func(context.Context, struct{}) error {
	return func(ctx context.Context, _ struct{}) error {
		_, err := publicexport.Export(ctx, activityService, geocodeService, outDir, publicexport.ExportOptions{Privacy: privacy})
		return err
	}
}

// ensureSchedule creates a schedule for kind if one doesn't already exist,
// so restarting iroha-job never inserts a duplicate tb_job_schedules row.
func ensureSchedule(db *gorm.DB, kind, scheduleKind, scheduleExpr string) error {
	var existing models.JobSchedule
	err := db.Where("kind = ?", kind).First(&existing).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("check existing schedule for %s: %w", kind, err)
	}

	now := time.Now().UTC()
	svc := jobs.NewService(db, nil, nil)
	_, err = svc.CreateSchedule(jobs.ScheduleInput{
		Kind:         kind,
		ScheduleKind: scheduleKind,
		ScheduleExpr: scheduleExpr,
		NextRunAt:    &now,
		Enabled:      true,
	})
	if err != nil {
		return fmt.Errorf("create schedule for %s: %w", kind, err)
	}
	return nil
}
