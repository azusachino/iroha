package daily

import (
	"encoding/base64"
	"errors"
	"sort"
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

type AggregateFilters struct {
	From        *time.Time
	To          *time.Time
	Granularity string
}

// AggregateBucket is one period's rollup. Ring fields are per-day averages over
// real ring days; Metrics holds a per-day average per metric slug (steps,
// vitals, …) so new metrics need no API change — same open-ended shape as
// tb_daily_metrics.
type AggregateBucket struct {
	Period         time.Time          `json:"period"`
	Days           int                `json:"days"`
	MoveKcalAvg    float64            `json:"move_kcal_avg"`
	ExerciseMinAvg float64            `json:"exercise_min_avg"`
	StandHoursAvg  float64            `json:"stand_hours_avg"`
	MoveClosedPct  float64            `json:"move_closed_pct"`
	Metrics        map[string]float64 `json:"metrics"`
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

// Aggregates rolls daily rows up per month/year. Rings and metrics live in
// different tables (tb_daily_summaries vs the long tb_daily_metrics), so this
// merges three grouped queries by period: ring averages + move-ring-closed
// share, per-metric averages, and a distinct-day count across both tables.
func (s *Service) Aggregates(filters AggregateFilters) ([]AggregateBucket, error) {
	period := "date_trunc('month', day::timestamp)"
	if filters.Granularity == "year" {
		period = "date_trunc('year', day::timestamp)"
	}
	applyRange := func(q *gorm.DB) *gorm.DB {
		if filters.From != nil {
			q = q.Where("day >= ?", *filters.From)
		}
		if filters.To != nil {
			q = q.Where("day <= ?", *filters.To)
		}
		return q
	}

	// Q1: ring averages — only real ring days live in tb_daily_summaries.
	type ringRow struct {
		Period      time.Time
		MoveAvg     float64
		ExerciseAvg float64
		StandAvg    float64
		RingDays    int
		MoveClosed  int
	}
	var ringRows []ringRow
	if err := applyRange(s.db.Table("tb_daily_summaries")).
		Select(period + ` as period,
			coalesce(avg(move_kcal),0) as move_avg,
			coalesce(avg(exercise_min),0) as exercise_avg,
			coalesce(avg(stand_hours),0) as stand_avg,
			count(*)::int as ring_days,
			count(*) filter (where move_goal_kcal > 0 and move_kcal >= move_goal_kcal)::int as move_closed`).
		Group("period").Scan(&ringRows).Error; err != nil {
		return nil, err
	}

	// Q2: per-metric per-day averages from the long metrics table.
	type metricRow struct {
		Period time.Time
		Metric string
		Avg    float64
	}
	var metricRows []metricRow
	if err := applyRange(s.db.Table("tb_daily_metrics")).
		Select(period + ` as period, metric, coalesce(avg(value),0) as avg`).
		Group("period, metric").Scan(&metricRows).Error; err != nil {
		return nil, err
	}

	// Q3: distinct calendar days per period across both tables.
	type dayRow struct {
		Period time.Time
		Days   int
	}
	var dayRows []dayRow
	if err := applyRange(s.db.Table(`(
		select day from tb_daily_summaries
		union
		select day from tb_daily_metrics
	) as d`)).
		Select(period + ` as period, count(distinct day)::int as days`).
		Group("period").Scan(&dayRows).Error; err != nil {
		return nil, err
	}

	byKey := map[int64]*AggregateBucket{}
	var order []int64
	getb := func(p time.Time) *AggregateBucket {
		k := p.UnixNano()
		b, ok := byKey[k]
		if !ok {
			b = &AggregateBucket{Period: p, Metrics: map[string]float64{}}
			byKey[k] = b
			order = append(order, k)
		}
		return b
	}
	for _, r := range ringRows {
		b := getb(r.Period)
		b.MoveKcalAvg = r.MoveAvg
		b.ExerciseMinAvg = r.ExerciseAvg
		b.StandHoursAvg = r.StandAvg
		if r.RingDays > 0 {
			b.MoveClosedPct = float64(r.MoveClosed) / float64(r.RingDays) * 100
		}
	}
	for _, r := range dayRows {
		getb(r.Period).Days = r.Days
	}
	for _, r := range metricRows {
		getb(r.Period).Metrics[r.Metric] = r.Avg
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })
	buckets := make([]AggregateBucket, 0, len(order))
	for _, k := range order {
		buckets = append(buckets, *byKey[k])
	}
	return buckets, nil
}
