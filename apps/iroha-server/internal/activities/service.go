package activities

import (
	"errors"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/internal/ids"
	"github.com/azusachino/iroha/apps/iroha-server/internal/models"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

type ListFilters struct {
	SportType   string
	StartedFrom *time.Time
	StartedTo   *time.Time
	Limit       int
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(filters ListFilters) ([]models.Activity, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := s.db.Model(&models.Activity{})
	if filters.SportType != "" {
		query = query.Where("sport_type = ?", filters.SportType)
	}
	if filters.StartedFrom != nil {
		query = query.Where("started_at >= ?", *filters.StartedFrom)
	}
	if filters.StartedTo != nil {
		query = query.Where("started_at <= ?", *filters.StartedTo)
	}

	var rows []models.Activity
	err := query.Order("started_at desc").Limit(limit).Find(&rows).Error
	return rows, err
}

func (s *Service) Get(id string) (models.Activity, bool, error) {
	decoded, err := ids.Decode(ids.ActivityPrefix, id)
	if err != nil {
		return models.Activity{}, false, err
	}

	var activity models.Activity
	err = s.db.First(&activity, "id = ?", decoded).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Activity{}, false, nil
	}
	if err != nil {
		return models.Activity{}, false, err
	}
	return activity, true, nil
}

func (s *Service) Route(id string) ([]models.ActivityRoutePoint, bool, error) {
	activity, found, err := s.Get(id)
	if err != nil || !found {
		return nil, found, err
	}

	var points []models.ActivityRoutePoint
	err = s.db.Select(
		"activity_id",
		"seq",
		"ts",
		"lat",
		"lon",
		"elevation_m",
		"distance_m",
		"speed_mps",
		"heart_rate",
	).Where("activity_id = ?", activity.ID).Order("seq asc").Find(&points).Error
	return points, true, err
}

func (s *Service) Samplings(id string) ([]models.ActivitySampling, bool, error) {
	activity, found, err := s.Get(id)
	if err != nil || !found {
		return nil, found, err
	}

	var samplings []models.ActivitySampling
	err = s.db.Where("activity_id = ?", activity.ID).Order("sampling_type asc, ts asc").Find(&samplings).Error
	return samplings, true, err
}

func (s *Service) Laps(id string) ([]models.ActivityLap, bool, error) {
	activity, found, err := s.Get(id)
	if err != nil || !found {
		return nil, found, err
	}

	var laps []models.ActivityLap
	err = s.db.Where("activity_id = ?", activity.ID).Order("lap_no asc").Find(&laps).Error
	return laps, true, err
}
