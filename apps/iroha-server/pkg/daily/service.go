package daily

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultPageLimit = 50

var ErrInvalidCursor = errors.New("invalid cursor")

type Cursor struct {
	Day time.Time
	ID  uuid.UUID
}

func EncodeCursor(cursor Cursor) string {
	raw := cursor.Day.UTC().Format("2006-01-02") + "|" + cursor.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(value string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	dayValue, idValue, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return Cursor{}, ErrInvalidCursor
	}
	day, err := time.Parse("2006-01-02", dayValue)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	id, err := uuid.Parse(idValue)
	if err != nil {
		return Cursor{}, ErrInvalidCursor
	}
	return Cursor{Day: day, ID: id}, nil
}

type ListFilters struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Cursor *Cursor
}

type Row struct {
	models.DailySummary
	Steps      *float64
	DistanceKM *float64
	Flights    *float64
}

type Page struct {
	Items      []Row
	NextCursor *Cursor
	HasMore    bool
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(filters ListFilters) (Page, error) {
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultPageLimit
	}

	query := s.db.Table("tb_daily_summaries as s").
		Select(`s.id, s.day, s.move_kcal, s.move_goal_kcal,
			s.exercise_min, s.exercise_goal_min, s.stand_hours,
			s.stand_goal_hours, s.source, s.first_raw_file_id,
			s.created_at, s.updated_at,
			steps.value as steps, distance.value as distance_km, flights.value as flights`).
		Joins("left join tb_daily_metrics as steps on steps.day = s.day and steps.metric = 'steps'").
		Joins("left join tb_daily_metrics as distance on distance.day = s.day and distance.metric = 'distance_km'").
		Joins("left join tb_daily_metrics as flights on flights.day = s.day and flights.metric = 'flights'")
	if filters.From != nil {
		query = query.Where("s.day >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("s.day <= ?", *filters.To)
	}
	if filters.Cursor != nil {
		query = query.Where("(s.day, s.id) < (?, ?)", filters.Cursor.Day, filters.Cursor.ID)
	}

	var rows []Row
	if err := query.Order("s.day desc, s.id desc").Limit(limit + 1).Scan(&rows).Error; err != nil {
		return Page{}, err
	}
	page := Page{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{Day: last.Day, ID: last.ID}
	}
	return page, nil
}
