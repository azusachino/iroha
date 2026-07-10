package activities

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/azusachino/iroha/apps/iroha-server/pkg/ids"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultPageLimit = 50

	// routeTrimMeters is how much distance is trimmed from the start and end
	// of each route before it is exposed publicly, so a viewer can't pinpoint
	// exact home/work addresses from where tracks begin or end. Distance is
	// measured along the track from the coordinates (route points carry no
	// reliable distance_m), so this trim is independent of import source.
	routeTrimMeters = 200
	// routeMinPoints is the minimum number of points a trimmed route must
	// retain to be worth emitting; shorter remainders are dropped entirely.
	routeMinPoints = 2
	// routeMaxPoints caps how many points each public route line carries,
	// keeping the response small; points are decimated evenly.
	routeMaxPoints = 150
	// earthRadiusMeters is the mean Earth radius used for haversine distance.
	earthRadiusMeters = 6371000
	// privateZoneRadiusMeters is the radius of an auto-detected private zone:
	// every route point within this distance of a frequent start/end hub
	// (home, work, gym) is dropped, and routes are split (not bridged) across
	// the gap, so the location is masked even where routes pass it mid-track.
	privateZoneRadiusMeters = 300
	// privateZoneMinCluster is how many start/end anchors must fall within a
	// radius of each other before that location is treated as a sensitive hub
	// worth masking (below this, the location is not frequent enough to matter).
	privateZoneMinCluster = 3
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
	SportType    string
	StartedFrom  *time.Time
	StartedTo    *time.Time
	DistanceMinM *float64
	DistanceMaxM *float64
	Limit        int
	Cursor       *Cursor
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
	return page, nil
}

// SummaryTotals holds aggregate metrics across a set of activities.
// DurationS is elapsed time; MovingTimeS is often unset (e.g. Apple imports),
// so DurationS is the reliable "total time" for a rollup.
type SummaryTotals struct {
	ActivityCount int     `json:"activity_count"`
	DistanceM     float64 `json:"distance_m"`
	DurationS     int     `json:"duration_s"`
	MovingTimeS   int     `json:"moving_time_s"`
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
	"coalesce(sum(duration_s), 0) as duration_s, " +
	"coalesce(sum(moving_time_s), 0) as moving_time_s"

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

	return Summary{Totals: totals, ByYear: byYear, ByMonth: byMonth, BySport: bySport}, nil
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

// RouteLine is one activity's simplified, privacy-trimmed polyline, suitable
// for the public "all routes" map. Points are [lon, lat] pairs (GeoJSON
// coordinate order).
type RouteLine struct {
	SportType string
	Year      string
	Points    [][2]float64
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

// RouteLines returns all activities' routes as privacy-masked polylines for
// the public map. Two layers of protection apply: each track's first/last
// routeTrimMeters are trimmed, and every point within an auto-detected private
// zone (a frequent start/end hub — home, work, gym) is dropped, splitting the
// line rather than bridging across it. A single activity may yield several
// polylines; segments below routeMinPoints are omitted, each decimated to
// routeMaxPoints.
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
		sport  string
		year   string
		coords [][2]float64
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
			current = track{sport: row.SportType, year: row.Year}
			currentID = row.ActivityID
		}
		current.coords = append(current.coords, [2]float64{*row.Lon, *row.Lat})
	}
	flush()

	// Auto-detect private zones from where activities frequently start/end,
	// then mask them across every route — endpoint trimming alone leaves these
	// hubs exposed where other routes pass near them mid-track.
	anchors := make([][2]float64, 0, len(tracks)*2)
	for _, t := range tracks {
		anchors = append(anchors, t.coords[0], t.coords[len(t.coords)-1])
	}
	zones := detectPrivateZones(anchors)

	lines := make([]RouteLine, 0, len(tracks))
	for _, t := range tracks {
		trimmed := trimRouteEnds(t.coords)
		if len(trimmed) < routeMinPoints {
			continue
		}
		// Decimate first, then mask, so masking runs over few points per route.
		for _, segment := range maskPrivateZones(decimatePoints(trimmed), zones) {
			if len(segment) < routeMinPoints {
				continue
			}
			lines = append(lines, RouteLine{SportType: t.sport, Year: t.year, Points: segment})
		}
	}
	return lines, nil
}

// detectPrivateZones returns every start/end anchor that sits in a dense
// cluster (>= privateZoneMinCluster anchors within privateZoneRadiusMeters),
// i.e. a frequent hub — home, work, gym. All such anchors are kept (not
// collapsed to a centroid) so masking covers the whole footprint of each hub,
// including its edges, rather than a single point.
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

// maskPrivateZones splits a track into the contiguous segments lying OUTSIDE
// all private zones. Points within privateZoneRadiusMeters of any zone are
// dropped and break the line, so no segment bridges straight across a hub.
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

// trimRouteEnds drops points within routeTrimMeters of the start and end of
// the track. Distance is accumulated along the track with the haversine
// formula, since route points carry no reliable distance_m. Tracks shorter
// than 2*routeTrimMeters are dropped entirely (nothing left after trimming).
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
