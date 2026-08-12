package activities

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
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
		query = query.Where("started_at <= ?", *filters.StartedTo)
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
type SummaryTotals struct {
	ActivityCount        int     `json:"activity_count"`
	DistanceM            float64 `json:"distance_m"`
	DistanceKnownCount   int     `json:"distance_known_count"`
	DistanceUnknownCount int     `json:"distance_unknown_count"`
	DurationS            int     `json:"duration_s"`
	ElevationGainM       float64 `json:"elevation_gain_m"`
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

const summaryMetrics = "count(*) as activity_count, " +
	"coalesce(sum(distance_m), 0) as distance_m, " +
	"count(*) filter (where distance_m is not null) as distance_known_count, " +
	"count(*) filter (where distance_m is null) as distance_unknown_count, " +
	"coalesce(sum(duration_s), 0) as duration_s, " +
	"coalesce(sum(elevation_gain_m), 0) as elevation_gain_m"

// Summary computes aggregate totals overall and grouped by year, month, and
// sport. Year/month are derived in the database session timezone (approximate
// for the public rollup; not tied to each activity's own timezone).
func (s *Service) Summary(year, sport string) (Summary, error) {
	base := func() *gorm.DB { return s.db.Model(&models.Activity{}) }
	filtered := func() *gorm.DB {
		db := base()
		if year != "" {
			db = db.Where("to_char(started_at, 'YYYY') = ?", year)
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

	var byYear []SummaryBucket
	if err := base().
		Select("extract(year from started_at)::text as key, " + summaryMetrics).
		Group("key").Order("key desc").Scan(&byYear).Error; err != nil {
		return Summary{}, fmt.Errorf("summary by year: %w", err)
	}

	var byMonth []SummaryBucket
	if err := filtered().
		Select("to_char(started_at, 'YYYY-MM') as key, " + summaryMetrics).
		Group("key").Order("key desc").Scan(&byMonth).Error; err != nil {
		return Summary{}, fmt.Errorf("summary by month: %w", err)
	}

	var bySport []SummaryBucket
	if err := filtered().
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
		yearKey := activity.StartedAt.Format("2006")
		monthKey := activity.StartedAt.Format("2006-01")
		if i, ok := byYearIndex[yearKey]; ok {
			byYear[i].DistanceM += *activity.DistanceM
			byYear[i].DistanceKnownCount++
			byYear[i].DistanceUnknownCount--
		}
		if (year == "" || yearKey == year) && (sport == "" || activity.SportType == sport) {
			if i, ok := byMonthIndex[monthKey]; ok {
				byMonth[i].DistanceM += *activity.DistanceM
				byMonth[i].DistanceKnownCount++
				byMonth[i].DistanceUnknownCount--
			}
			if i, ok := bySportIndex[activity.SportType]; ok {
				bySport[i].DistanceM += *activity.DistanceM
				bySport[i].DistanceKnownCount++
				bySport[i].DistanceUnknownCount--
			}
			totals.DistanceKnownCount++
			totals.DistanceUnknownCount--
			totals.DistanceM += *activity.DistanceM
		}
	}
	for _, activity := range missingElevations {
		if activity.ElevationGainM == nil {
			continue
		}
		yearKey := activity.StartedAt.Format("2006")
		monthKey := activity.StartedAt.Format("2006-01")
		if i, ok := byYearIndex[yearKey]; ok {
			byYear[i].ElevationGainM += *activity.ElevationGainM
		}
		if (year == "" || yearKey == year) && (sport == "" || activity.SportType == sport) {
			if i, ok := byMonthIndex[monthKey]; ok {
				byMonth[i].ElevationGainM += *activity.ElevationGainM
			}
			if i, ok := bySportIndex[activity.SportType]; ok {
				bySport[i].ElevationGainM += *activity.ElevationGainM
			}
			totals.ElevationGainM += *activity.ElevationGainM
		}
	}

	return Summary{Totals: totals, ByYear: byYear, ByMonth: byMonth, BySport: bySport}, nil
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
	if !filters.From.Before(filters.To) {
		return SummaryTotals{}, errors.New("period from must be before to")
	}
	location := time.UTC
	if filters.Timezone != "" {
		var err error
		location, err = time.LoadLocation(filters.Timezone)
		if err != nil {
			return SummaryTotals{}, fmt.Errorf("load timezone: %w", err)
		}
	}
	from := filters.From.In(location).UTC()
	to := filters.To.In(location).UTC()

	var rows []models.Activity
	if err := s.db.Where("started_at >= ? AND started_at < ?", from, to).Find(&rows).Error; err != nil {
		return SummaryTotals{}, err
	}
	if err := s.hydrateSwimmingDistances(rows); err != nil {
		return SummaryTotals{}, fmt.Errorf("period swimming distances: %w", err)
	}
	if err := s.hydrateElevationGain(rows); err != nil {
		return SummaryTotals{}, fmt.Errorf("period elevation gain: %w", err)
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
	hydrated := []models.Activity{activity}
	if err := s.hydrateSwimmingDistances(hydrated); err != nil {
		return models.Activity{}, false, fmt.Errorf("hydrate swimming distance: %w", err)
	}
	if err := s.hydrateElevationGain(hydrated); err != nil {
		return models.Activity{}, false, fmt.Errorf("hydrate elevation gain: %w", err)
	}
	if err := s.hydrateMovingTime(hydrated); err != nil {
		return models.Activity{}, false, fmt.Errorf("hydrate moving time: %w", err)
	}
	return hydrated[0], true, nil
}

type swimmingDistancePoint struct {
	ActivityID uuid.UUID
	Seq        int
	Lat        *float64
	Lon        *float64
}

// hydrateSwimmingDistances fills only missing swim distances from their GPS
// trace. The imported row remains unchanged; this is a read-model correction
// for open-water workouts whose source did not provide a pool-style total.
func (s *Service) hydrateSwimmingDistances(activities []models.Activity) error {
	ids := make([]uuid.UUID, 0)
	for _, activity := range activities {
		if activity.DistanceM == nil && isSwimming(activity.SportType) {
			ids = append(ids, activity.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var rows []swimmingDistancePoint
	if err := s.db.Table("tb_activity_route_points").
		Select("activity_id", "seq", "lat", "lon").
		Where("activity_id IN ?", ids).
		Order("activity_id, seq asc").
		Find(&rows).Error; err != nil {
		return err
	}

	tracks := make(map[uuid.UUID][][2]float64, len(ids))
	for _, row := range rows {
		if row.Lat == nil || row.Lon == nil {
			continue
		}
		tracks[row.ActivityID] = append(tracks[row.ActivityID], [2]float64{*row.Lon, *row.Lat})
	}
	for i := range activities {
		if distance := routeDistanceMeters(tracks[activities[i].ID]); distance > 0 {
			activities[i].DistanceM = &distance
		}
	}
	return nil
}

func isSwimming(sport string) bool {
	return strings.Contains(strings.ToLower(sport), "swim")
}

func routeDistanceMeters(coords [][2]float64) float64 {
	var distance float64
	for i := 1; i < len(coords); i++ {
		distance += haversineMeters(coords[i-1], coords[i])
	}
	return distance
}

type elevationRoutePoint struct {
	ActivityID uuid.UUID
	Seq        int
	ElevationM *float64
}

// hydrateElevationGain fills only missing elevation gain from the GPS trace's
// per-point elevation samples, the same read-model-correction pattern as
// hydrateSwimmingDistances -- sources that omit a workout-level elevation
// summary (seen on Apple Health imports) often still carry per-point
// elevation on the route itself.
func (s *Service) hydrateElevationGain(activities []models.Activity) error {
	ids := make([]uuid.UUID, 0)
	for _, activity := range activities {
		if activity.ElevationGainM == nil {
			ids = append(ids, activity.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var rows []elevationRoutePoint
	if err := s.db.Table("tb_activity_route_points").
		Select("activity_id", "seq", "elevation_m").
		Where("activity_id IN ?", ids).
		Order("activity_id, seq asc").
		Find(&rows).Error; err != nil {
		return err
	}

	tracks := make(map[uuid.UUID][]float64, len(ids))
	for _, row := range rows {
		if row.ElevationM == nil {
			continue
		}
		tracks[row.ActivityID] = append(tracks[row.ActivityID], *row.ElevationM)
	}
	for i := range activities {
		// len, not gain > 0: a flat route (gain computes to exactly 0) is a
		// real, valid answer worth showing as "0 m" -- only the absence of
		// any GPS elevation samples means there's nothing to report.
		if points := tracks[activities[i].ID]; len(points) > 0 {
			gain := routeElevationGainMeters(points)
			activities[i].ElevationGainM = &gain
		}
	}
	return nil
}

// elevationNoiseFloorMeters discards per-step elevation deltas below this
// threshold before summing gain -- consumer GPS elevation is noisy enough
// (typically several meters) that summing every up-tick without a floor
// wildly overstates gain on a route that is actually flat.
const elevationNoiseFloorMeters = 2.0

type movingTimeRoutePoint struct {
	ActivityID uuid.UUID
	Seq        int
	Ts         *time.Time
	Lat        *float64
	Lon        *float64
}

type timedPoint struct {
	ts       time.Time
	lon, lat float64
}

// movingSpeedThresholdMPS is the implied point-to-point speed below which an
// interval counts as stopped (waiting at a light, a break) rather than
// moving -- about 1.8 km/h, well under even a slow walk.
const movingSpeedThresholdMPS = 0.5

// hydrateMovingTime fills only missing moving time by summing GPS-trace
// intervals whose implied speed clears movingSpeedThresholdMPS -- the same
// read-model-correction pattern as hydrateSwimmingDistances/
// hydrateElevationGain. Apple Health exports in particular often report
// elapsed duration but never a separate moving-time figure.
func (s *Service) hydrateMovingTime(activities []models.Activity) error {
	ids := make([]uuid.UUID, 0)
	for _, activity := range activities {
		if activity.MovingTimeS == nil {
			ids = append(ids, activity.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	var rows []movingTimeRoutePoint
	if err := s.db.Table("tb_activity_route_points").
		Select("activity_id", "seq", "ts", "lat", "lon").
		Where("activity_id IN ?", ids).
		Order("activity_id, seq asc").
		Find(&rows).Error; err != nil {
		return err
	}

	tracks := make(map[uuid.UUID][]timedPoint, len(ids))
	for _, row := range rows {
		if row.Ts == nil || row.Lat == nil || row.Lon == nil {
			continue
		}
		tracks[row.ActivityID] = append(tracks[row.ActivityID], timedPoint{ts: *row.Ts, lon: *row.Lon, lat: *row.Lat})
	}
	for i := range activities {
		if points := tracks[activities[i].ID]; len(points) > 1 {
			seconds := routeMovingTimeSeconds(points)
			// The GPS trace's own timestamp span can run a couple of seconds
			// past the source's reported duration (a stray point either
			// side of the official start/stop); moving time can never
			// exceed elapsed time, so clamp rather than show a contradiction.
			if d := activities[i].DurationS; d != nil && seconds > *d {
				seconds = *d
			}
			activities[i].MovingTimeS = &seconds
		}
	}
	return nil
}

func routeMovingTimeSeconds(points []timedPoint) int {
	var seconds float64
	for i := 1; i < len(points); i++ {
		dt := points[i].ts.Sub(points[i-1].ts).Seconds()
		if dt <= 0 {
			continue
		}
		dist := haversineMeters([2]float64{points[i-1].lon, points[i-1].lat}, [2]float64{points[i].lon, points[i].lat})
		if dist/dt >= movingSpeedThresholdMPS {
			seconds += dt
		}
	}
	return int(seconds)
}

func routeElevationGainMeters(elevations []float64) float64 {
	var gain float64
	for i := 1; i < len(elevations); i++ {
		if delta := elevations[i] - elevations[i-1]; delta > elevationNoiseFloorMeters {
			gain += delta
		}
	}
	return gain
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

// RouteLine is one activity's simplified public polyline. Points are [lon,
// lat] pairs (GeoJSON coordinate order).
type RouteLine struct {
	ActivityID uuid.UUID
	SportType  string
	Year       string
	Points     [][2]float64
}

// routePointRow is the minimal projection needed to build public route
// lines: no timestamps, elevation, speed, or heart rate leave the database.
type routePointRow struct {
	ActivityID uuid.UUID
	SportType  string
	Year       string
	Seq        int
	Lat        *float64
	Lon        *float64
}

// RouteLines returns all activities' routes as decimated polylines for the
// public map. The public projection intentionally preserves the complete
// track; decimation only keeps the static snapshot reasonably sized.
func (s *Service) RouteLines() ([]RouteLine, error) {
	var rows []routePointRow
	err := s.db.Table("tb_activity_route_points AS p").
		Select("p.activity_id", "a.sport_type", "to_char(a.started_at, 'YYYY') as year", "p.seq", "p.lat", "p.lon").
		Joins("JOIN tb_activities AS a ON a.id = p.activity_id").
		Order("p.activity_id, p.seq asc").
		Find(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("route lines: %w", err)
	}

	// Group ordered points into one coordinate track per activity.
	type track struct {
		activityID uuid.UUID
		sport      string
		year       string
		coords     [][2]float64
	}
	var tracks []track
	var currentID uuid.UUID
	var current track
	flush := func() {
		if len(current.coords) > 0 {
			tracks = append(tracks, current)
		}
	}
	for _, row := range rows {
		if row.Lat == nil || row.Lon == nil {
			continue
		}
		if row.ActivityID != currentID {
			flush()
			current = track{activityID: row.ActivityID, sport: row.SportType, year: row.Year}
			currentID = row.ActivityID
		}
		current.coords = append(current.coords, [2]float64{*row.Lon, *row.Lat})
	}
	flush()

	lines := make([]RouteLine, 0, len(tracks))
	for _, t := range tracks {
		if len(t.coords) < routeMinPoints {
			continue
		}
		lines = append(lines, RouteLine{
			ActivityID: t.activityID,
			SportType:  t.sport,
			Year:       t.year,
			Points:     decimatePoints(t.coords),
		})
	}
	return lines, nil
}

func detectPrivateZones(anchors [][2]float64) [][2]float64 {
	zones := make([][2]float64, 0)
	for i := range anchors {
		count := 0
		for j := range anchors {
			if haversineMeters(anchors[i], anchors[j]) <= privateZoneRadiusMeters {
				count++
			}
		}
		if count >= privateZoneMinCluster {
			zones = append(zones, anchors[i])
		}
	}
	return zones
}

func maskPrivateZones(coords, zones [][2]float64) [][][2]float64 {
	inZone := func(p [2]float64) bool {
		for _, z := range zones {
			if haversineMeters(p, z) <= privateZoneRadiusMeters {
				return true
			}
		}
		return false
	}
	var segments [][][2]float64
	var current [][2]float64
	for _, coord := range coords {
		if inZone(coord) {
			if len(current) > 0 {
				segments = append(segments, current)
				current = nil
			}
			continue
		}
		current = append(current, coord)
	}
	if len(current) > 0 {
		segments = append(segments, current)
	}
	return segments
}

func trimRouteEnds(coords [][2]float64) [][2]float64 {
	if len(coords) < routeMinPoints {
		return nil
	}
	cumulative := make([]float64, len(coords))
	for i := 1; i < len(coords); i++ {
		cumulative[i] = cumulative[i-1] + haversineMeters(coords[i-1], coords[i])
	}
	total := cumulative[len(cumulative)-1]
	if total <= 2*routeTrimMeters {
		return nil
	}
	hi := total - routeTrimMeters
	trimmed := make([][2]float64, 0, len(coords))
	for i, coord := range coords {
		if cumulative[i] >= routeTrimMeters && cumulative[i] <= hi {
			trimmed = append(trimmed, coord)
		}
	}
	return trimmed
}

// haversineMeters returns the great-circle distance between two [lon, lat]
// points in meters.
func haversineMeters(a, b [2]float64) float64 {
	lat1 := a[1] * math.Pi / 180
	lat2 := b[1] * math.Pi / 180
	dLat := lat2 - lat1
	dLon := (b[0] - a[0]) * math.Pi / 180

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadiusMeters * math.Asin(math.Sqrt(h))
}

// decimatePoints keeps at most routeMaxPoints points, chosen with an even
// stride, always including the final point so the line still reaches the
// track's (trimmed) end.
func decimatePoints(points [][2]float64) [][2]float64 {
	if len(points) <= routeMaxPoints {
		return points
	}

	stride := int(math.Ceil(float64(len(points)) / float64(routeMaxPoints)))
	out := make([][2]float64, 0, routeMaxPoints+1)
	for i := 0; i < len(points); i += stride {
		out = append(out, points[i])
	}
	last := points[len(points)-1]
	if out[len(out)-1] != last {
		out = append(out, last)
	}
	return out
}

// Samplings returns an activity's measurement stream. When types are given the
// result is limited to those sampling_types — the detail view only charts
// heart_rate, so it avoids pulling the (much larger) power/energy/speed streams.
func (s *Service) Samplings(id string, types ...string) ([]models.ActivitySampling, bool, error) {
	activity, found, err := s.Get(id)
	if err != nil || !found {
		return nil, found, err
	}

	q := s.db.Where("activity_id = ?", activity.ID)
	if len(types) > 0 {
		q = q.Where("sampling_type IN ?", types)
	}

	var samplings []models.ActivitySampling
	err = q.Order("sampling_type asc, ts asc").Find(&samplings).Error
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
