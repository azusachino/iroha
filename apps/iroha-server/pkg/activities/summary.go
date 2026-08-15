package activities

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"gorm.io/gorm"
)

type SummaryTotals struct {
	ActivityCount        int     `json:"activity_count"`
	DistanceM            float64 `json:"distance_m"`
	DistanceKnownCount   int     `json:"distance_known_count"`
	DistanceUnknownCount int     `json:"distance_unknown_count"`
	DurationS            int     `json:"duration_s"`
	ElevationGainM       float64 `json:"elevation_gain_m"`
}

type PeriodSportTotal struct {
	Sport              string
	ActivityCount      int
	DistanceM          float64
	DistanceKnownCount int
	DurationS          int
}

type PeriodReport struct {
	Totals  SummaryTotals
	BySport []PeriodSportTotal
}

// SummaryBucket is one grouped total, keyed by year or sport type.
type SummaryBucket struct {
	Key string `json:"key"`
	SummaryTotals
}

// Summary is a derived, aggregate-only view suitable for the public page.
// The time-bucketed rollups (ByYear / ByMonth) are the generally useful shape
// across data types; BySport is the activity-specific facet.
type Summary struct {
	Totals  SummaryTotals   `json:"totals"`
	ByYear  []SummaryBucket `json:"by_year"`
	ByMonth []SummaryBucket `json:"by_month"`
	BySport []SummaryBucket `json:"by_sport"`
}

type ActiveDay struct {
	Day           string `json:"day"`
	ActivityCount int    `json:"activity_count"`
}

// Overview is the server-owned projection used by the cockpit dashboard. It
// keeps the heatmap and recent list from having to sweep the activity ledger.
// Recent remains as canonical rows; the HTTP layer is responsible for its
// private DTO mapping.
type Overview struct {
	Summary       Summary
	ActiveDays    []ActiveDay
	Recent        []models.Activity
	CurrentStreak int
}

const summaryMetrics = "count(*) as activity_count, " +
	"coalesce(sum(distance_m), 0) as distance_m, " +
	"count(*) filter (where distance_m is not null) as distance_known_count, " +
	"count(*) filter (where distance_m is null) as distance_unknown_count, " +
	"coalesce(sum(duration_s), 0) as duration_s, " +
	"coalesce(sum(elevation_gain_m), 0) as elevation_gain_m"

// Summary computes aggregate totals in UTC for callers without an explicit
// display timezone. HTTP callers should use SummaryInTimezone so calendar
// buckets match the user's canonical period controls.
func (s *Service) Summary(year, sport string) (Summary, error) {
	return s.SummaryInTimezone(year, sport, "UTC")
}

// SummaryInTimezone computes aggregate totals grouped in the requested IANA
// timezone. The activity instants remain canonical; only their calendar
// presentation bucket changes.
func (s *Service) SummaryInTimezone(year, sport, timezone string) (Summary, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Summary{}, fmt.Errorf("load timezone: %w", err)
	}
	zone := location.String()
	base := func() *gorm.DB { return s.db.Model(&models.Activity{}) }
	filtered := func() *gorm.DB {
		db := base()
		if year != "" {
			db = db.Where("to_char(started_at AT TIME ZONE ?, 'YYYY') = ?", zone, year)
		}
		if sport != "" {
			db = db.Where("sport_type = ?", sport)
		}
		return db
	}

	var totals SummaryTotals
	if err := filtered().Select(summaryMetrics).Scan(&totals).Error; err != nil {
		return Summary{}, fmt.Errorf("summary totals: %w", err)
	}

	byYearScope := base()
	if sport != "" {
		byYearScope = byYearScope.Where("sport_type = ?", sport)
	}
	var byYear []SummaryBucket
	if err := byYearScope.
		Select("to_char(started_at AT TIME ZONE ?, 'YYYY') as key, "+summaryMetrics, zone).
		Group("key").Order("key desc").Scan(&byYear).Error; err != nil {
		return Summary{}, fmt.Errorf("summary by year: %w", err)
	}

	var byMonth []SummaryBucket
	if err := filtered().
		Select("to_char(started_at AT TIME ZONE ?, 'YYYY-MM') as key, "+summaryMetrics, zone).
		Group("key").Order("key desc").Scan(&byMonth).Error; err != nil {
		return Summary{}, fmt.Errorf("summary by month: %w", err)
	}

	bySportScope := base()
	if year != "" {
		bySportScope = bySportScope.Where("to_char(started_at AT TIME ZONE ?, 'YYYY') = ?", zone, year)
	}
	var bySport []SummaryBucket
	if err := bySportScope.
		Select("sport_type as key, " + summaryMetrics).
		Group("sport_type").Order("activity_count desc").Scan(&bySport).Error; err != nil {
		return Summary{}, fmt.Errorf("summary by sport: %w", err)
	}

	// GORM's Scan leaves the destination nil when zero rows match, which
	// marshals to JSON null instead of [] and crashes any frontend for-of/map
	// over an empty year/month/sport breakdown.
	if byYear == nil {
		byYear = []SummaryBucket{}
	}
	if byMonth == nil {
		byMonth = []SummaryBucket{}
	}
	if bySport == nil {
		bySport = []SummaryBucket{}
	}

	// Apple Health open-water swims may have no source distance even though
	// their route points are available. Keep the SQL aggregate fast for the
	// normal case, then add those read-model distances to every affected rollup.
	var missingSwimDistances []models.Activity
	if err := s.db.Model(&models.Activity{}).
		Where("distance_m IS NULL").
		Where("sport_type ILIKE ?", "%swim%").
		Find(&missingSwimDistances).Error; err != nil {
		return Summary{}, fmt.Errorf("summary swimming distances: %w", err)
	}
	if err := s.hydrateSwimmingDistances(missingSwimDistances); err != nil {
		return Summary{}, fmt.Errorf("summary swimming distances: %w", err)
	}
	var missingElevations []models.Activity
	if err := s.db.Model(&models.Activity{}).
		Where("elevation_gain_m IS NULL").
		Find(&missingElevations).Error; err != nil {
		return Summary{}, fmt.Errorf("summary elevation gain: %w", err)
	}
	if err := s.hydrateElevationGain(missingElevations); err != nil {
		return Summary{}, fmt.Errorf("summary elevation gain: %w", err)
	}

	byYearIndex := make(map[string]int, len(byYear))
	for i := range byYear {
		byYearIndex[byYear[i].Key] = i
	}
	byMonthIndex := make(map[string]int, len(byMonth))
	for i := range byMonth {
		byMonthIndex[byMonth[i].Key] = i
	}
	bySportIndex := make(map[string]int, len(bySport))
	for i := range bySport {
		bySportIndex[bySport[i].Key] = i
	}
	for _, activity := range missingSwimDistances {
		if activity.DistanceM == nil {
			continue
		}
		localStartedAt := activity.StartedAt.In(location)
		yearKey := localStartedAt.Format("2006")
		monthKey := localStartedAt.Format("2006-01")
		inYear := year == "" || yearKey == year
		inSport := sport == "" || activity.SportType == sport
		if inSport {
			if i, ok := byYearIndex[yearKey]; ok {
				byYear[i].DistanceM += *activity.DistanceM
				byYear[i].DistanceKnownCount++
				byYear[i].DistanceUnknownCount--
			}
		}
		if inYear && inSport {
			if i, ok := byMonthIndex[monthKey]; ok {
				byMonth[i].DistanceM += *activity.DistanceM
				byMonth[i].DistanceKnownCount++
				byMonth[i].DistanceUnknownCount--
			}
		}
		if inYear {
			if i, ok := bySportIndex[activity.SportType]; ok {
				bySport[i].DistanceM += *activity.DistanceM
				bySport[i].DistanceKnownCount++
				bySport[i].DistanceUnknownCount--
			}
		}
		if inYear && inSport {
			totals.DistanceKnownCount++
			totals.DistanceUnknownCount--
			totals.DistanceM += *activity.DistanceM
		}
	}
	for _, activity := range missingElevations {
		if activity.ElevationGainM == nil {
			continue
		}
		localStartedAt := activity.StartedAt.In(location)
		yearKey := localStartedAt.Format("2006")
		monthKey := localStartedAt.Format("2006-01")
		inYear := year == "" || yearKey == year
		inSport := sport == "" || activity.SportType == sport
		if inSport {
			if i, ok := byYearIndex[yearKey]; ok {
				byYear[i].ElevationGainM += *activity.ElevationGainM
			}
		}
		if inYear && inSport {
			if i, ok := byMonthIndex[monthKey]; ok {
				byMonth[i].ElevationGainM += *activity.ElevationGainM
			}
		}
		if inYear {
			if i, ok := bySportIndex[activity.SportType]; ok {
				bySport[i].ElevationGainM += *activity.ElevationGainM
			}
		}
		if inYear && inSport {
			totals.ElevationGainM += *activity.ElevationGainM
		}
	}

	return Summary{Totals: totals, ByYear: byYear, ByMonth: byMonth, BySport: bySport}, nil
}

func (s *Service) Overview(timezone string, recentLimit int) (Overview, error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return Overview{}, fmt.Errorf("load timezone: %w", err)
	}
	if recentLimit <= 0 || recentLimit > 100 {
		recentLimit = 5
	}

	summary, err := s.SummaryInTimezone("", "", timezone)
	if err != nil {
		return Overview{}, err
	}
	recent, err := s.List(ListFilters{Limit: recentLimit})
	if err != nil {
		return Overview{}, fmt.Errorf("overview recent activities: %w", err)
	}

	var activeDays []ActiveDay
	if err := s.db.Model(&models.Activity{}).
		Select("to_char(started_at AT TIME ZONE ?, 'YYYY-MM-DD') as day, count(*) as activity_count", location.String()).
		Group("day").Order("day desc").Scan(&activeDays).Error; err != nil {
		return Overview{}, fmt.Errorf("overview active days: %w", err)
	}
	if activeDays == nil {
		activeDays = []ActiveDay{}
	}

	active := make(map[string]struct{}, len(activeDays))
	for _, day := range activeDays {
		active[day.Day] = struct{}{}
	}
	streak := 0
	for day := time.Now().In(location); ; day = day.AddDate(0, 0, -1) {
		if _, ok := active[day.Format("2006-01-02")]; !ok {
			break
		}
		streak++
	}

	return Overview{
		Summary:       summary,
		ActiveDays:    activeDays,
		Recent:        recent.Items,
		CurrentStreak: streak,
	}, nil
}

// Bounds returns the earliest and latest calendar dates with a real
// activity record, in the requested timezone. maxDate is capped at now so a
// bad-import row with a future timestamp can never widen the navigable
// range; the past is never capped. ok is false when no activities exist.
func (s *Service) Bounds(now time.Time, timezone string) (minDate, maxDate string, ok bool, err error) {
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return "", "", false, fmt.Errorf("load timezone: %w", err)
	}
	zone := location.String()
	var row struct {
		Min *string
		Max *string
	}
	if err := s.db.Model(&models.Activity{}).
		Select("to_char(min(started_at) AT TIME ZONE ?, 'YYYY-MM-DD') as min, to_char(max(started_at) AT TIME ZONE ?, 'YYYY-MM-DD') as max", zone, zone).
		Scan(&row).Error; err != nil {
		return "", "", false, fmt.Errorf("activity bounds: %w", err)
	}
	if row.Min == nil || row.Max == nil {
		return "", "", false, nil
	}
	nowDate := now.In(location).Format("2006-01-02")
	maxDate = *row.Max
	if maxDate > nowDate {
		maxDate = nowDate
	}
	if *row.Min > maxDate {
		// Every real row is dated after now (test fixtures or bad imports) --
		// there is no usable historical range yet.
		return "", "", false, nil
	}
	return *row.Min, maxDate, true, nil
}

// PeriodFilters defines an activity report window. From is inclusive and To
// is exclusive. Callers provide calendar boundaries in Timezone; converting
// them to UTC before querying keeps membership stable across database session
// timezones.
type PeriodFilters struct {
	From     time.Time
	To       time.Time
	Timezone string
}

// PeriodSummary returns corrected totals for one requested-timezone window.
// It is deliberately separate from Summary, whose year/month buckets retain
// the legacy public-export shape.
func (s *Service) PeriodSummary(filters PeriodFilters) (SummaryTotals, error) {
	rows, err := s.periodActivities(filters)
	if err != nil {
		return SummaryTotals{}, err
	}

	var totals SummaryTotals
	for _, activity := range rows {
		totals.ActivityCount++
		if activity.DistanceM == nil {
			totals.DistanceUnknownCount++
		} else {
			totals.DistanceKnownCount++
			totals.DistanceM += *activity.DistanceM
		}
		if activity.DurationS != nil {
			totals.DurationS += *activity.DurationS
		}
		if activity.ElevationGainM != nil {
			totals.ElevationGainM += *activity.ElevationGainM
		}
	}
	return totals, nil
}

// PeriodActivities exposes the selected canonical activity rows to server-side
// metric resolvers. The resolver owns aggregation; this method only applies the
// existing timezone-aware period membership and hydration rules.
func (s *Service) PeriodActivities(filters PeriodFilters) ([]models.Activity, error) {
	return s.periodActivities(filters)
}

func (s *Service) PeriodReport(filters PeriodFilters) (PeriodReport, error) {
	rows, err := s.periodActivities(filters)
	if err != nil {
		return PeriodReport{}, err
	}

	result := PeriodReport{BySport: make([]PeriodSportTotal, 0)}
	bySport := make(map[string]*PeriodSportTotal)
	for _, activity := range rows {
		result.Totals.ActivityCount++
		if activity.DistanceM == nil {
			result.Totals.DistanceUnknownCount++
		} else {
			result.Totals.DistanceKnownCount++
			result.Totals.DistanceM += *activity.DistanceM
		}
		if activity.DurationS != nil {
			result.Totals.DurationS += *activity.DurationS
		}
		total := bySport[activity.SportType]
		if total == nil {
			total = &PeriodSportTotal{Sport: activity.SportType}
			bySport[activity.SportType] = total
			result.BySport = append(result.BySport, *total)
		}
		total.ActivityCount++
		if activity.DistanceM != nil {
			total.DistanceKnownCount++
			total.DistanceM += *activity.DistanceM
		}
		if activity.DurationS != nil {
			total.DurationS += *activity.DurationS
		}
	}
	for i := range result.BySport {
		result.BySport[i] = *bySport[result.BySport[i].Sport]
	}
	sort.Slice(result.BySport, func(i, j int) bool { return result.BySport[i].Sport < result.BySport[j].Sport })
	return result, nil
}

func (s *Service) periodActivities(filters PeriodFilters) ([]models.Activity, error) {
	if !filters.From.Before(filters.To) {
		return nil, errors.New("period from must be before to")
	}
	location := time.UTC
	if filters.Timezone != "" {
		var err error
		location, err = time.LoadLocation(filters.Timezone)
		if err != nil {
			return nil, fmt.Errorf("load timezone: %w", err)
		}
	}
	from := filters.From.In(location).UTC()
	to := filters.To.In(location).UTC()

	var rows []models.Activity
	if err := s.db.Where("started_at >= ? AND started_at < ?", from, to).Find(&rows).Error; err != nil {
		return nil, err
	}
	if err := s.hydrateSwimmingDistances(rows); err != nil {
		return nil, fmt.Errorf("period swimming distances: %w", err)
	}
	if err := s.hydrateElevationGain(rows); err != nil {
		return nil, fmt.Errorf("period elevation gain: %w", err)
	}
	return rows, nil
}
