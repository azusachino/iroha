package imports

import (
	"errors"
	"fmt"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	syncRunStatusRunning   = "running"
	syncRunStatusCompleted = "completed"
	syncRunStatusFailed    = "failed"
	syncRunLease           = 30 * time.Minute
)

var ErrSyncAlreadyRunning = errors.New("media sync is already running")

func (s *SyncRunner) beginSyncRun(connectorID string) (models.MediaSyncRun, error) {
	if connectorID == "" {
		return models.MediaSyncRun{}, fmt.Errorf("connector id is required")
	}
	now := time.Now().UTC()
	runID, err := ids.New()
	if err != nil {
		return models.MediaSyncRun{}, err
	}
	run := models.MediaSyncRun{ID: runID, ConnectorID: connectorID, Status: syncRunStatusRunning, StartedAt: now, CreatedAt: now, UpdatedAt: now}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("select pg_advisory_xact_lock(hashtext(?))", "iroha:media-sync:"+connectorID).Error; err != nil {
			return err
		}
		var active models.MediaSyncRun
		result := tx.Where("connector_id = ? and status = ?", connectorID, syncRunStatusRunning).First(&active)
		if result.Error == nil {
			if now.Sub(active.UpdatedAt) < syncRunLease {
				return ErrSyncAlreadyRunning
			}
			message := "sync lease expired"
			finishedAt := now
			if err := tx.Model(&models.MediaSyncRun{}).Where("id = ?", active.ID).Updates(map[string]any{
				"status":        syncRunStatusFailed,
				"error_message": &message,
				"finished_at":   &finishedAt,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
		} else if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		return tx.Create(&run).Error
	})
	if err != nil {
		return models.MediaSyncRun{}, err
	}
	return run, nil
}

func (s *SyncRunner) touchSyncRun(runID uuid.UUID) error {
	return s.db.Model(&models.MediaSyncRun{}).Where("id = ? and status = ?", runID, syncRunStatusRunning).Update("updated_at", time.Now().UTC()).Error
}

func (s *SyncRunner) completeSyncRun(runID uuid.UUID) error {
	now := time.Now().UTC()
	result := s.db.Model(&models.MediaSyncRun{}).Where("id = ? and status = ?", runID, syncRunStatusRunning).Updates(map[string]any{
		"status":      syncRunStatusCompleted,
		"finished_at": &now,
		"updated_at":  now,
	})
	return result.Error
}

func (s *SyncRunner) failSyncRun(runID uuid.UUID, cause error) error {
	if cause == nil {
		cause = errors.New("media sync failed")
	}
	message := cause.Error()
	now := time.Now().UTC()
	result := s.db.Model(&models.MediaSyncRun{}).Where("id = ? and status = ?", runID, syncRunStatusRunning).Updates(map[string]any{
		"status":        syncRunStatusFailed,
		"error_message": &message,
		"finished_at":   &now,
		"updated_at":    now,
	})
	return result.Error
}
