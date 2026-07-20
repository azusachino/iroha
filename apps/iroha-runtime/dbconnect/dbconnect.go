// Package dbconnect opens a Postgres connection with a bounded retry, so a
// process that starts slightly before its database is reachable (e.g. pod
// ordering during a k8s rollout) does not immediately exit.
package dbconnect

import (
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	maxAttempts   = 10
	initialWait   = 250 * time.Millisecond
	maxWait       = 5 * time.Second
	backoffFactor = 2
)

// Connect opens a Postgres connection, retrying with exponential backoff
// (capped at maxWait) up to maxAttempts before returning the last error.
func Connect(url string, cfg *gorm.Config, logger *slog.Logger) (*gorm.DB, error) {
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
