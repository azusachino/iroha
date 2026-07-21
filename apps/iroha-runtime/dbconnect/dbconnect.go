// Package dbconnect opens a Postgres connection with a bounded retry, so a
// process that starts slightly before its database is reachable (e.g. pod
// ordering during a k8s rollout) does not immediately exit.
package dbconnect

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	maxAttempts   = 10
	initialWait   = 250 * time.Millisecond
	maxWait       = 5 * time.Second
	backoffFactor = 2
	slowQuery     = 200 * time.Millisecond
)

// Connect opens a Postgres connection, retrying with exponential backoff
// (capped at maxWait) up to maxAttempts before returning the last error.
func Connect(url string, cfg *gorm.Config, logger *slog.Logger) (*gorm.DB, error) {
	if cfg.Logger == nil {
		cfg.Logger = &slogGormLogger{logger: logger}
	}

	wait := initialWait
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		db, err := gorm.Open(postgres.Open(url), cfg)
		if err == nil {
			return db, nil
		}
		lastErr = err

		if attempt == maxAttempts {
			break
		}
		logger.Warn("database not ready, retrying",
			"attempt", attempt, "max_attempts", maxAttempts, "wait", wait, "error", err)
		time.Sleep(wait)
		if wait *= backoffFactor; wait > maxWait {
			wait = maxWait
		}
	}

	return nil, lastErr
}

// slogGormLogger routes GORM's query logging through the app's structured
// logger instead of GORM's default ANSI-colored stdlib logger, so every log
// line in a pod shares one JSON format. Record-not-found is expected during
// existence lookups (e.g. import resolution) and is not logged as an error.
type slogGormLogger struct {
	logger *slog.Logger
}

func (l *slogGormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return l }

func (l *slogGormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.logger.InfoContext(ctx, msg, "args", args)
}

func (l *slogGormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.logger.WarnContext(ctx, msg, "args", args)
}

func (l *slogGormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.logger.ErrorContext(ctx, msg, "args", args)
}

func (l *slogGormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)

	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		sql, rows := fc()
		l.logger.ErrorContext(ctx, "db query failed", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds(), "error", err)
		return
	}
	if elapsed > slowQuery {
		sql, rows := fc()
		l.logger.WarnContext(ctx, "slow db query", "sql", sql, "rows", rows, "elapsed_ms", elapsed.Milliseconds())
	}
}
