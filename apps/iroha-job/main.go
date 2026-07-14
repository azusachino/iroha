package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
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
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/rawfiles"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	once := flag.Bool("once", false, "process at most one due schedule and one queued job")
	pollInterval := flag.Duration("poll-interval", time.Second, "idle backoff between polls when the queue is empty")
	concurrency := flag.Int("concurrency", 4, "number of worker goroutines draining the queue")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	cfg, err := config.Load("iroha.toml")
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(cfg.Database.URL), &gorm.Config{})
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

	cacheClient := cache.New(cfg.Cache.URL)
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
	var mediaBridge imports.MediaRefBridge
	if os.Getenv(config.EnvBangumiBridge) != "" || os.Getenv(config.EnvMALAniListBridge) != "" {
		bridge, err := imports.LoadTwoHopMediaRefBridge(os.Getenv(config.EnvBangumiBridge), os.Getenv(config.EnvMALAniListBridge))
		if err != nil {
			logger.Error("load media bridge", "error", err)
			os.Exit(1)
		}
		mediaBridge = bridge
	}
	importService := imports.NewServiceWithRegistryAndBridge(db, logger, parserVersion, enqueuer, cacheClient, providers, mediaBridge)
	rawFileService, err := rawfiles.NewService(db, cfg.Storage.DataDir)
	if err != nil {
		logger.Error("create raw file service", "error", err)
		os.Exit(1)
	}
	connectors, err := connectorregistry.New(
		anilist.NewConnector(os.Getenv(config.EnvAniListUsername), os.Getenv(config.EnvAniListToken)),
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

	jobsService = jobs.NewService(db, logger, registry.Handlers())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
		return importService.Process(importJobID)
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
		return runner.Run(ctx, payload.ConnectorID, payload.Credentials)
	}
}
