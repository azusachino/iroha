package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	imports "github.com/azusachino/iroha/apps/iroha-imports"
	"github.com/azusachino/iroha/apps/iroha-runtime/cache"
	"github.com/azusachino/iroha/apps/iroha-runtime/config"
	"github.com/azusachino/iroha/apps/iroha-runtime/dbconnect"
	"github.com/azusachino/iroha/apps/iroha-runtime/jobs"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-runtime/rawfiles"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/daily"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/expenses"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/geocode"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/httpapi"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/media"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/mediaresolution"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metrics"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/metricseries"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/sleep"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/tasks"
	"gorm.io/gorm"
)

func main() {
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

	rawFileService, err := rawfiles.NewService(db, cfg.Storage.DataDir)
	if err != nil {
		logger.Error("create raw file service", "error", err)
		os.Exit(1)
	}
	// parser_version identifies the parser build; a completed import at a
	// different version triggers a reprocess (purge + re-persist) rather than
	// a duplicate append. Overridable via IROHA_PARSER_VERSION so it can be
	// bumped without recompiling.
	parserVersion := os.Getenv("IROHA_PARSER_VERSION")
	if parserVersion == "" {
		parserVersion = imports.DefaultParserVersion
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

	jobsService := jobs.NewService(db, logger, nil)
	enqueuer := &jobEnqueuer{jobsService: jobsService}
	importService := imports.NewService(db, logger, parserVersion, enqueuer, cacheClient)
	geocodeService := geocode.NewService(db, enqueuer, cacheClient)
	activityService := activities.NewService(db)
	sleepService := sleep.NewService(db)
	dailyService := daily.NewService(db)
	expenseService := expenses.NewService(db)
	metricRegistry, err := metrics.DefaultRegistry()
	if err != nil {
		logger.Error("create metric registry", "error", err)
		os.Exit(1)
	}
	metricSeriesService := metricseries.NewService(
		metricRegistry,
		metricseries.DailyServiceSource{Service: dailyService},
		metricseries.ActivityServiceSource{Service: activityService},
		metricseries.ExpenseServiceSource{Service: expenseService},
	)
	mediaService := media.NewService(db)
	mediaResolutionService := mediaresolution.NewService(db)
	taskService := tasks.NewService(db)
	briefingRegistry, err := httpapi.NewBriefingRegistry(dailyService, sleepService, activityService, mediaService)
	if err != nil {
		logger.Error("create briefing registry", "error", err)
		os.Exit(1)
	}

	server := httpapi.NewServer(httpapi.Dependencies{
		Config:                 cfg,
		Logger:                 logger,
		ActivityService:        activityService,
		SleepService:           sleepService,
		DailyService:           dailyService,
		ExpenseService:         expenseService,
		MediaService:           mediaService,
		MediaResolutionService: mediaResolutionService,
		MetricRegistry:         metricRegistry,
		MetricSeriesService:    metricSeriesService,
		BriefingRegistry:       briefingRegistry,
		ImportService:          importService,
		RawFileService:         rawFileService,
		Cache:                  cacheClient,
		GeocodeService:         geocodeService,
		JobEnqueuer:            enqueuer,
		JobsService:            jobsService,
		TaskService:            taskService,
		MaxUploadBytes:         2 << 30,
		AllowedOrigins:         cfg.Server.AllowedOrigins,
	})

	logger.Info("starting iroha-server", "addr", cfg.Server.Addr)
	httpServer := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           server,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Minute,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received, draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("server shut down cleanly")
	}
}

type jobEnqueuer struct {
	jobsService *jobs.Service
}

func (e *jobEnqueuer) EnqueueTx(tx *gorm.DB, kind string, payload any) (models.Job, error) {
	return e.jobsService.EnqueueTx(tx, jobs.EnqueueInput{
		Kind:    kind,
		Payload: payload,
	})
}
