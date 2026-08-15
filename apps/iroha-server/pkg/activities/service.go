package activities

import (
	"fmt"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultPageLimit = 50
	routeTrimMeters  = 200

	// routeMinPoints is the minimum number of points a trimmed route must
	// retain to be worth emitting; shorter remainders are dropped entirely.
	routeMinPoints = 2
	// routeMaxPoints caps how many points each public route line carries,
	// keeping the response small; points are decimated evenly.
	routeMaxPoints = 150
	// earthRadiusMeters is the mean Earth radius used for haversine distance.
	earthRadiusMeters       = 6371000
	privateZoneRadiusMeters = 300
	privateZoneMinCluster   = 3
)

type Service struct {
	db *gorm.DB
}

// Cursor is a keyset position over the (started_at desc, id desc) ordering.
type Cursor struct {
	StartedAt time.Time
	ID        uuid.UUID
}

type ListFilters struct {
	SportType     string
	StartedFrom   *time.Time
	StartedTo     *time.Time
	StartedBefore *time.Time
	DistanceMinM  *float64
	DistanceMaxM  *float64
	Limit         int
	Cursor        *Cursor
}

// Page is one keyset window; NextCursor is nil when no further rows exist.
type Page struct {
	Items      []models.Activity
	NextCursor *Cursor
	HasMore    bool
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(filters ListFilters) (Page, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}

	query := s.db.Model(&models.Activity{})
	if filters.SportType != "" {
		query = query.Where("sport_type = ?", filters.SportType)
	}
	if filters.StartedFrom != nil {
		query = query.Where("started_at >= ?", *filters.StartedFrom)
	}
	if filters.StartedTo != nil {
		query = query.Where("started_at < ?", *filters.StartedTo)
	}
	if filters.StartedBefore != nil {
		query = query.Where("started_at < ?", *filters.StartedBefore)
	}
	// Distance filters naturally exclude rows with a NULL distance_m.
	if filters.DistanceMinM != nil {
		query = query.Where("distance_m >= ?", *filters.DistanceMinM)
	}
	if filters.DistanceMaxM != nil {
		query = query.Where("distance_m <= ?", *filters.DistanceMaxM)
	}
	if filters.Cursor != nil {
		// Row-value comparison walks the (started_at desc, id desc) order:
		// keep rows strictly after the cursor position.
		query = query.Where("(started_at, id) < (?, ?)", filters.Cursor.StartedAt, filters.Cursor.ID)
	}

	// Fetch one extra row to detect whether another page follows.
	var rows []models.Activity
	if err := query.Order("started_at desc, id desc").Limit(limit + 1).Find(&rows).Error; err != nil {
		return Page{}, err
	}

	page := Page{Items: rows}
	if len(rows) > limit {
		last := rows[limit-1]
		page.Items = rows[:limit]
		page.HasMore = true
		page.NextCursor = &Cursor{StartedAt: last.StartedAt, ID: last.ID}
	}
	if err := s.hydrateSwimmingDistances(page.Items); err != nil {
		return Page{}, fmt.Errorf("hydrate swimming distances: %w", err)
	}
	if err := s.hydrateElevationGain(page.Items); err != nil {
		return Page{}, fmt.Errorf("hydrate elevation gain: %w", err)
	}
	if err := s.hydrateMovingTime(page.Items); err != nil {
		return Page{}, fmt.Errorf("hydrate moving time: %w", err)
	}
	return page, nil
}

// SummaryTotals holds aggregate metrics across a set of activities. Distance
// coverage is explicit because a NULL source distance is not zero distance.
