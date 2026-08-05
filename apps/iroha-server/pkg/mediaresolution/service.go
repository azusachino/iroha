// Package mediaresolution serves the media-sync resolution inbox --
// dedupe-candidate and progress-conflict tasks that apps/iroha-imports'
// resolver writes to tb_media_resolution_tasks but never had an API or UI
// surface to read from. Resolving a task here only records the operator's
// decision in resolution_json; it does not merge media rows or apply a
// progress choice -- consuming that decision is a separate, not-yet-built
// job, same as the rest of the resolver's cross-provider dedup being a
// documented "later" item (docs/media-sync-connectors.md).
package mediaresolution

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusOpen      = "open"
	StatusResolved  = "resolved"
	StatusDismissed = "dismissed"

	defaultLimit = 50
)

var ErrNotFound = gorm.ErrRecordNotFound

type ListFilters struct {
	Status string
	Limit  int
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// List defaults to StatusOpen -- this is an inbox, so the useful view is
// "what's waiting," not the full history. Pass an explicit status (including
// "resolved" or "dismissed") to see closed tasks instead.
func (s *Service) List(filters ListFilters) ([]models.MediaResolutionTask, error) {
	status := filters.Status
	if status == "" {
		status = StatusOpen
	}
	limit := filters.Limit
	if limit <= 0 || limit > 200 {
		limit = defaultLimit
	}

	var rows []models.MediaResolutionTask
	err := s.db.Model(&models.MediaResolutionTask{}).
		Where("status = ?", status).
		Order("created_at asc").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}

// Resolve closes an open task with the operator's decision. resolution may
// be nil for a plain dismissal; when non-nil it is stored verbatim as the
// task's resolution_json for a future consumer to act on.
func (s *Service) Resolve(id uuid.UUID, status string, resolution json.RawMessage) (models.MediaResolutionTask, error) {
	if status != StatusResolved && status != StatusDismissed {
		return models.MediaResolutionTask{}, errors.New("status must be resolved or dismissed")
	}
	if len(resolution) == 0 {
		resolution = json.RawMessage(`{}`)
	}

	now := time.Now().UTC()
	result := s.db.Model(&models.MediaResolutionTask{}).
		Where("id = ? AND status = ?", id, StatusOpen).
		Updates(map[string]any{
			"status":          status,
			"resolution_json": resolution,
			"resolved_at":     now,
		})
	if result.Error != nil {
		return models.MediaResolutionTask{}, result.Error
	}
	if result.RowsAffected == 0 {
		return models.MediaResolutionTask{}, ErrNotFound
	}

	var task models.MediaResolutionTask
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return models.MediaResolutionTask{}, err
	}
	return task, nil
}
