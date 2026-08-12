package sleep

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const defaultPageLimit = 50

type Cursor struct {
	WakeDate time.Time
	ID       uuid.UUID
}

func EncodeCursor(cursor Cursor) string {
	raw := cursor.WakeDate.UTC().Format("2006-01-02") + "|" + cursor.ID.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(value string) (Cursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	wakeDateValue, idValue, ok := strings.Cut(string(decoded), "|")
	if !ok {
		return Cursor{}, errors.New("invalid cursor")
	}
	wakeDate, err := time.Parse("2006-01-02", wakeDateValue)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(idValue)
	if err != nil {
		return Cursor{}, errors.New("invalid cursor")
	}
	return Cursor{WakeDate: wakeDate, ID: id}, nil
}

type ListFilters struct {
	From   *time.Time
	To     *time.Time
	Limit  int
	Cursor *Cursor
}

type Page struct {
	Items      []models.SleepSession
	NextCursor *Cursor
	HasMore    bool
}

type AggregateFilters struct {
	From        *time.Time
	To          *time.Time
	Granularity string
}

type AggregateBucket struct {
	Period            time.Time `json:"period"`
	SessionCount      int       `json:"session_count"`
	MainSleepCount    int       `json:"main_sleep_count"`
	NapCount          int       `json:"nap_count"`
	AverageAsleepS    float64   `json:"average_asleep_s"`
	AverageTimeInBedS float64   `json:"average_time_in_bed_s"`
	AverageEfficiency float64   `json:"average_efficiency"`
	CoreS             int       `json:"core_s"`
	DeepS             int       `json:"deep_s"`
	RemS              int       `json:"rem_s"`
	AwakeS            int       `json:"awake_s"`
	UnspecifiedS      int       `json:"unspecified_s"`
}

type PeriodReport struct {
	SessionCount      int
	MainSleepCount    int
	NapCount          int
	AverageAsleepS    float64
	AverageTimeInBedS float64
	AverageEfficiency float64
	StageSeconds      struct {
		Core        int
		Deep        int
		Rem         int
		Awake       int
		Unspecified int
	}
}

type PeriodFilters struct {
	From time.Time
	To   time.Time
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

	query := s.db.Model(&models.SleepSession{})
	if filters.From != nil {
		query = query.Where("wake_date >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("wake_date <= ?", *filters.To)
	}
	if filters.Cursor != nil {
		query = query.Where("(wake_date, id) < (?, ?)", filters.Cursor.WakeDate, filters.Cursor.ID)
	}

	var rows []models.SleepSession
	if err := query.Order("wake_date desc, id desc").Limit(limit + 1).Find(&rows).Error; err != nil {
		return Page{}, err
	}
	page := Page{Items: rows}
	if len(rows) > limit {
		page.Items = rows[:limit]
		page.HasMore = true
		last := page.Items[len(page.Items)-1]
		page.NextCursor = &Cursor{WakeDate: last.WakeDate, ID: last.ID}
	}
	return page, nil
}

func (s *Service) Aggregates(filters AggregateFilters) ([]AggregateBucket, error) {
	periodExpression := "date_trunc('month', wake_date::timestamp)"
	if filters.Granularity == "year" {
		periodExpression = "date_trunc('year', wake_date::timestamp)"
	}
	query := s.db.Model(&models.SleepSession{}).
		Select(periodExpression + ` as period,
			count(*)::int as session_count,
			count(*) filter (where is_main_sleep)::int as main_sleep_count,
			count(*) filter (where not is_main_sleep)::int as nap_count,
			coalesce(avg(asleep_s) filter (where is_main_sleep), 0) as average_asleep_s,
			coalesce(avg(time_in_bed_s) filter (where is_main_sleep), 0) as average_time_in_bed_s,
			coalesce(avg(efficiency) filter (where is_main_sleep), 0) as average_efficiency,
			coalesce(sum(core_s) filter (where is_main_sleep), 0)::int as core_s,
			coalesce(sum(deep_s) filter (where is_main_sleep), 0)::int as deep_s,
			coalesce(sum(rem_s) filter (where is_main_sleep), 0)::int as rem_s,
			coalesce(sum(awake_s) filter (where is_main_sleep), 0)::int as awake_s,
			coalesce(sum(unspecified_s) filter (where is_main_sleep), 0)::int as unspecified_s`).
		Group("period").Order("period asc")
	if filters.From != nil {
		query = query.Where("wake_date >= ?", *filters.From)
	}
	if filters.To != nil {
		query = query.Where("wake_date <= ?", *filters.To)
	}
	var buckets []AggregateBucket
	if err := query.Scan(&buckets).Error; err != nil {
		return nil, err
	}
	return buckets, nil
}

func (s *Service) PeriodReport(filters PeriodFilters) (PeriodReport, error) {
	if !filters.From.Before(filters.To) {
		return PeriodReport{}, errors.New("period from must be before to")
	}
	from := time.Date(filters.From.UTC().Year(), filters.From.UTC().Month(), filters.From.UTC().Day(), 0, 0, 0, 0, time.UTC)
	to := time.Date(filters.To.UTC().Year(), filters.To.UTC().Month(), filters.To.UTC().Day(), 0, 0, 0, 0, time.UTC)
	var rows []models.SleepSession
	if err := s.db.Where("wake_date >= ? and wake_date < ?", from, to).Find(&rows).Error; err != nil {
		return PeriodReport{}, err
	}
	result := PeriodReport{SessionCount: len(rows)}
	for _, row := range rows {
		if !row.IsMainSleep {
			result.NapCount++
			continue
		}
		result.MainSleepCount++
		result.AverageAsleepS += float64(row.AsleepS)
		result.AverageTimeInBedS += float64(row.TimeInBedS)
		result.AverageEfficiency += row.Efficiency
		result.StageSeconds.Core += row.CoreS
		result.StageSeconds.Deep += row.DeepS
		result.StageSeconds.Rem += row.RemS
		result.StageSeconds.Awake += row.AwakeS
		result.StageSeconds.Unspecified += row.UnspecifiedS
	}
	if result.MainSleepCount > 0 {
		count := float64(result.MainSleepCount)
		result.AverageAsleepS /= count
		result.AverageTimeInBedS /= count
		result.AverageEfficiency /= count
	}
	return result, nil
}

func (s *Service) Get(id string) (models.SleepSession, bool, error) {
	decoded, err := ids.Decode(ids.SleepPrefix, id)
	if err != nil {
		return models.SleepSession{}, false, err
	}
	var session models.SleepSession
	err = s.db.First(&session, "id = ?", decoded).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.SleepSession{}, false, nil
	}
	if err != nil {
		return models.SleepSession{}, false, err
	}
	return session, true, nil
}

func (s *Service) Segments(id string) ([]models.SleepSegment, bool, error) {
	session, found, err := s.Get(id)
	if err != nil || !found {
		return nil, found, err
	}
	var segments []models.SleepSegment
	err = s.db.Where("session_id = ?", session.ID).Order("seq asc").Find(&segments).Error
	return segments, true, err
}
