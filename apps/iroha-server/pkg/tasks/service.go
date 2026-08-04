package tasks

import (
	"errors"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	StatusOpen      = "open"
	StatusCompleted = "completed"
	StatusCanceled  = "canceled"
	defaultLimit    = 50
)

var (
	ErrEmptyTitle = errors.New("task title is required")
	ErrNotFound   = gorm.ErrRecordNotFound
)

type ListFilters struct {
	Status string
	DueOn  *time.Time
	Limit  int
}

type CreateInput struct {
	Title    string
	Notes    string
	DueDate  *time.Time
	Priority int
	Source   string
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func NormalizeTitle(value string) string {
	return strings.TrimSpace(value)
}

func (s *Service) List(filters ListFilters) ([]models.Task, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultLimit
	}

	query := s.db.Model(&models.Task{})
	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}
	if filters.DueOn != nil {
		query = query.Where("due_date = ?", filters.DueOn.UTC().Format("2006-01-02"))
	}

	var result []models.Task
	if err := query.Order("case when status = 'open' then 0 else 1 end, due_date asc nulls last, priority desc, created_at desc").Limit(limit).Find(&result).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) Create(input CreateInput) (models.Task, error) {
	title := NormalizeTitle(input.Title)
	if title == "" {
		return models.Task{}, ErrEmptyTitle
	}

	id, err := ids.New()
	if err != nil {
		return models.Task{}, err
	}
	now := time.Now().UTC()
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "web"
	}
	task := models.Task{
		ID: id, Title: title, Notes: strings.TrimSpace(input.Notes), Status: StatusOpen,
		DueDate: input.DueDate, Priority: input.Priority, Source: source,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.db.Create(&task).Error; err != nil {
		return models.Task{}, err
	}
	return task, nil
}

func (s *Service) Complete(id uuid.UUID) (models.Task, error) {
	return s.setStatus(id, StatusCompleted)
}

func (s *Service) Cancel(id uuid.UUID) (models.Task, error) {
	return s.setStatus(id, StatusCanceled)
}

func (s *Service) setStatus(id uuid.UUID, status string) (models.Task, error) {
	now := time.Now().UTC()
	result := s.db.Model(&models.Task{}).Where("id = ? and status = ?", id, StatusOpen).Updates(map[string]any{
		"status": status, "completed_at": completedAt(status, now), "updated_at": now,
	})
	if result.Error != nil {
		return models.Task{}, result.Error
	}
	if result.RowsAffected == 0 {
		return models.Task{}, ErrNotFound
	}
	var task models.Task
	if err := s.db.First(&task, "id = ?", id).Error; err != nil {
		return models.Task{}, err
	}
	return task, nil
}

func completedAt(status string, now time.Time) *time.Time {
	if status != StatusCompleted {
		return nil
	}
	return &now
}
