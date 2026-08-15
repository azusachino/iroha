package activities

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
