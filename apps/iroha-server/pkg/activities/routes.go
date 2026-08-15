package activities

import (
	"fmt"
	"math"

	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
)

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
