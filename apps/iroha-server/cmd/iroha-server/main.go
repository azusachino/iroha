package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/azusachino/iroha/apps/iroha-server/internal/activities"
	"github.com/azusachino/iroha/apps/iroha-server/internal/config"
	"github.com/azusachino/iroha/apps/iroha-server/internal/httpapi"
	"github.com/azusachino/iroha/apps/iroha-server/internal/imports"
	"github.com/azusachino/iroha/apps/iroha-server/internal/rawfiles"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
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

	rawFileService, err := rawfiles.NewService(db, cfg.Storage.DataDir)
	if err != nil {
		logger.Error("create raw file service", "error", err)
		os.Exit(1)
	}
	// parser_version bumped from "dev" to force the new Apple Health pipeline
	// (task-8 of iroha:apple-health-fidelity) to run instead of short-circuiting
	// via the reuse guard against the pre-refactor "dev" completed import.
	importService := imports.NewService(db, logger, "apple-health-2026-07")
	activityService := activities.NewService(db)

	server := httpapi.NewServer(httpapi.Dependencies{
		Config:          cfg,
		Logger:          logger,
		ActivityService: activityService,
		ImportService:   importService,
		RawFileService:  rawFileService,
		MaxUploadBytes:  2 << 30,
		AllowedOrigins:  nil,
	})

	logger.Info("starting iroha-server", "addr", cfg.Server.Addr)
	if err := http.ListenAndServe(cfg.Server.Addr, server); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
