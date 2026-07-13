package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	coreimports "github.com/azusachino/iroha/apps/iroha-core/imports"
	"github.com/azusachino/iroha/apps/iroha-imports"
	providerregistry "github.com/azusachino/iroha/apps/iroha-providers/registry"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	once := flag.Bool("once", false, "process at most one due schedule and one queued job")
	pollInterval := flag.Duration("poll-interval", 5*time.Second, "delay between polling attempts")
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

	enqueuer := &noopEnqueuer{}
	providers, err := providerregistry.New()
	if err != nil {
		logger.Error("build provider registry", "error", err)
		os.Exit(1)
	}
	importService := imports.NewServiceWithRegistry(db, logger, parserVersion, enqueuer, cacheClient, providers)

	// Both kinds run the same import pipeline (Process dispatches on the
	// job's parser_kind); only the parser kinds parsers.IsImplemented allows
	// are ever enqueued, so only those get a handler here.
	importHandler := makeImportParseHandler(importService)
	handlers := map[string]jobs.Handler{
		jobs.KindAppleImportParse: importHandler,
		jobs.KindGPXImportParse:   importHandler,
	}

	jobsService := jobs.NewService(db, logger, handlers)
	ctx := context.Background()

	for {
		if err := tick(ctx, jobsService, logger, workerID); err != nil && !errors.Is(err, jobs.ErrNoJobAvailable) {
			logger.Error("worker tick", "error", err)
		}
		if *once {
			return
		}
		time.Sleep(*pollInterval)
	}
}

func tick(ctx context.Context, service *jobs.Service, logger *slog.Logger, workerID string) error {
	enqueued, err := service.EnqueueDueSchedules(1)
	if err != nil {
		return err
	}
	if enqueued > 0 {
		logger.Info("enqueued due schedules", "count", enqueued)
	}

	job, err := service.ProcessNext(ctx, workerID)
	if err != nil {
		return err
	}
	logger.Info("processed job", "job_id", job.ID.String(), "kind", job.Kind)
	return nil
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

type noopEnqueuer struct{}

func (n *noopEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	return models.Job{}, nil
}

func makeImportParseHandler(importService *imports.Service) jobs.Handler {
	return func(ctx context.Context, job models.Job) error {
		var payload struct {
			ImportJobID string `json:"import_job_id"`
		}
		if err := json.Unmarshal(job.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode payload: %w", err)
		}
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
