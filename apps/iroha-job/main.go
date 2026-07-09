package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/config"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/jobs"
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

	service := jobs.NewService(db, logger, nil)
	ctx := context.Background()

	for {
		if err := tick(ctx, service, logger, workerID); err != nil && !errors.Is(err, jobs.ErrNoJobAvailable) {
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
