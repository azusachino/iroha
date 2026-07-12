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
	Steps           *float64
	DistanceKM      *float64
	Flights         *float64
	RestingHR       *float64
	WalkingHRAvg    *float64
	HRVSDNN         *float64 `gorm:"column:hrv_sdnn"`
	SpO2Avg         *float64
	SpO2Min         *float64
	RespiratoryRate *float64
	VO2Max          *float64 `gorm:"column:vo2max"`
	BodyMassKG      *float64
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

	query := s.db.Table(`(
		select day from tb_daily_summaries
		union
		select day from tb_daily_metrics
	) as days`).
		Select(`coalesce(s.id, anchor.id) as id, days.day,
			coalesce(s.move_kcal, 0) as move_kcal,
			coalesce(s.move_goal_kcal, 0) as move_goal_kcal,
			coalesce(s.exercise_min, 0) as exercise_min,
			coalesce(s.exercise_goal_min, 0) as exercise_goal_min,
			coalesce(s.stand_hours, 0) as stand_hours,
			coalesce(s.stand_goal_hours, 0) as stand_goal_hours,
			coalesce(s.source, anchor.source, '') as source,
			coalesce(s.first_raw_file_id, anchor.first_raw_file_id) as first_raw_file_id,
			coalesce(s.created_at, anchor.created_at) as created_at,
			coalesce(s.updated_at, anchor.updated_at) as updated_at,
			steps.value as steps, distance.value as distance_km, flights.value as flights,
			resting_hr.value as resting_hr, walking_hr_avg.value as walking_hr_avg,
			hrv_sdnn.value as hrv_sdnn, spo2_avg.value as spo2_avg,
			spo2_min.value as spo2_min, respiratory_rate.value as respiratory_rate,
			vo2max.value as vo2max, body_mass_kg.value as body_mass_kg`).
		Joins("left join tb_daily_summaries as s on s.day = days.day").
		Joins(`left join lateral (
			select id, first_raw_file_id, source, created_at, updated_at
			from tb_daily_metrics where day = days.day order by id limit 1
		) as anchor on true`).
		Joins("left join tb_daily_metrics as steps on steps.day = days.day and steps.metric = 'steps'").
		Joins("left join tb_daily_metrics as distance on distance.day = days.day and distance.metric = 'distance_km'").
		Joins("left join tb_daily_metrics as flights on flights.day = days.day and flights.metric = 'flights'").
		Joins("left join tb_daily_metrics as resting_hr on resting_hr.day = days.day and resting_hr.metric = 'resting_hr'").
		Joins("left join tb_daily_metrics as walking_hr_avg on walking_hr_avg.day = days.day and walking_hr_avg.metric = 'walking_hr_avg'").
		Joins("left join tb_daily_metrics as hrv_sdnn on hrv_sdnn.day = days.day and hrv_sdnn.metric = 'hrv_sdnn'").
		Joins("left join tb_daily_metrics as spo2_avg on spo2_avg.day = days.day and spo2_avg.metric = 'spo2_avg'").
		Joins("left join tb_daily_metrics as spo2_min on spo2_min.day = days.day and spo2_min.metric = 'spo2_min'").
		Joins("left join tb_daily_metrics as respiratory_rate on respiratory_rate.day = days.day and respiratory_rate.metric = 'respiratory_rate'").
		Joins("left join tb_daily_metrics as vo2max on vo2max.day = days.day and vo2max.metric = 'vo2max'").
		Joins("left join tb_daily_metrics as body_mass_kg on body_mass_kg.day = days.day and body_mass_kg.metric = 'body_mass_kg'")
	if filters.From != nil {
		query = query.Where("days.day >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("days.day <= ?", *filters.To)
	}
	if filters.Cursor != nil {
		query = query.Where("(days.day, coalesce(s.id, anchor.id)) < (?, ?)", filters.Cursor.Day, filters.Cursor.ID)
	}

	var rows []Row
	if err := query.Order("days.day desc, coalesce(s.id, anchor.id) desc").Limit(limit + 1).Scan(&rows).Error; err != nil {
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
