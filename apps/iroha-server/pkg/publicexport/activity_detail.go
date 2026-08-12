package publicexport

import (
	"fmt"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/activities"
)

// ApprovedActivityIDs is the explicit editorial allowlist for full public
// activity detail. All other activities keep the normal sanitized projection.
var ApprovedActivityIDs = map[string]struct{}{
	"act_019f82a5-87b2-7b31-9ebf-19f169899a76": {},
}

type ActivityDetail struct {
	Activity  ActivityDetailActivity     `json:"activity"`
	Route     []ActivityDetailRoutePoint `json:"route"`
	Samplings []ActivityDetailSampling   `json:"samplings"`
	Laps      []ActivityDetailLap        `json:"laps"`
}

type ActivityDetailActivity struct {
	Activity
	SourceKind string `json:"source_kind"`
}

type ActivityDetailRoutePoint struct {
	Seq        int        `json:"seq"`
	Ts         *time.Time `json:"ts,omitempty"`
	Lat        float64    `json:"lat"`
	Lon        float64    `json:"lon"`
	ElevationM *float64   `json:"elevation_m,omitempty"`
	DistanceM  *float64   `json:"distance_m,omitempty"`
	SpeedMPS   *float64   `json:"speed_mps,omitempty"`
	HeartRate  *int       `json:"heart_rate,omitempty"`
}

type ActivityDetailSampling struct {
	ID           string    `json:"id"`
	SamplingType string    `json:"sampling_type"`
	Ts           time.Time `json:"ts"`
	Value        float64   `json:"value"`
	Unit         string    `json:"unit"`
}

type ActivityDetailLap struct {
	ID            string     `json:"id"`
	LapNo         int        `json:"lap_no"`
	StartTs       *time.Time `json:"start_ts,omitempty"`
	EndTs         *time.Time `json:"end_ts,omitempty"`
	DistanceM     *float64   `json:"distance_m,omitempty"`
	DurationS     *int       `json:"duration_s,omitempty"`
	AvgHR         *int       `json:"avg_hr,omitempty"`
	AvgPaceSPerKM *float64   `json:"avg_pace_s_per_km,omitempty"`
}

func ApprovedActivityDetails(svc *activities.Service) (map[string]ActivityDetail, error) {
	details := make(map[string]ActivityDetail, len(ApprovedActivityIDs))
	for id := range ApprovedActivityIDs {
		activity, found, err := svc.Get(id)
		if err != nil {
			return nil, fmt.Errorf("get approved activity %s: %w", id, err)
		}
		if !found {
			return nil, fmt.Errorf("approved activity %s not found", id)
		}

		route, found, err := svc.Route(id)
		if err != nil {
			return nil, fmt.Errorf("get approved route %s: %w", id, err)
		}
		if !found {
			return nil, fmt.Errorf("approved route %s not found", id)
		}
		samplings, found, err := svc.Samplings(id, "heart_rate")
		if err != nil {
			return nil, fmt.Errorf("get approved samplings %s: %w", id, err)
		}
		if !found {
			return nil, fmt.Errorf("approved samplings %s not found", id)
		}
		laps, found, err := svc.Laps(id)
		if err != nil {
			return nil, fmt.Errorf("get approved laps %s: %w", id, err)
		}
		if !found {
			return nil, fmt.Errorf("approved laps %s not found", id)
		}

		details[id] = ActivityDetail{
			Activity: ActivityDetailActivity{
				Activity:   ToActivity(activity),
				SourceKind: activity.SourceKind,
			},
			Route:     toActivityDetailRoute(route),
			Samplings: toActivityDetailSamplings(samplings),
			Laps:      toActivityDetailLaps(laps),
		}
	}
	return details, nil
}

func toActivityDetailRoute(points []models.ActivityRoutePoint) []ActivityDetailRoutePoint {
	out := make([]ActivityDetailRoutePoint, 0, len(points))
	for _, point := range points {
		out = append(out, ActivityDetailRoutePoint{
			Seq:        point.Seq,
			Ts:         point.Ts,
			Lat:        point.Lat,
			Lon:        point.Lon,
			ElevationM: point.ElevationM,
			DistanceM:  point.DistanceM,
			SpeedMPS:   point.SpeedMPS,
			HeartRate:  point.HeartRate,
		})
	}
	return out
}

func toActivityDetailSamplings(points []models.ActivitySampling) []ActivityDetailSampling {
	out := make([]ActivityDetailSampling, 0, len(points))
	for _, point := range points {
		out = append(out, ActivityDetailSampling{
			ID:           ids.Encode("sampling", point.ID),
			SamplingType: point.SamplingType,
			Ts:           point.Ts,
			Value:        point.Value,
			Unit:         point.Unit,
		})
	}
	return out
}

func toActivityDetailLaps(laps []models.ActivityLap) []ActivityDetailLap {
	out := make([]ActivityDetailLap, 0, len(laps))
	for _, lap := range laps {
		out = append(out, ActivityDetailLap{
			ID:            ids.Encode("lap", lap.ID),
			LapNo:         lap.LapNo,
			StartTs:       lap.StartTs,
			EndTs:         lap.EndTs,
			DistanceM:     lap.DistanceM,
			DurationS:     lap.DurationS,
			AvgHR:         lap.AvgHR,
			AvgPaceSPerKM: lap.AvgPaceSPerKM,
		})
	}
	return out
}

func ValidateActivityDetails(details map[string]ActivityDetail) error {
	for id, detail := range details {
		if _, ok := ApprovedActivityIDs[id]; !ok {
			return fmt.Errorf("activity %s is not approved for full public detail", id)
		}
		if detail.Activity.ID != id || !strings.HasPrefix(detail.Activity.ID, ids.ActivityPrefix+"_") {
			return fmt.Errorf("activity %s has mismatched public id", id)
		}
		if err := validateActivity(detail.Activity.Activity); err != nil {
			return fmt.Errorf("activity detail %s: %w", id, err)
		}
		for _, point := range detail.Route {
			if point.Lat < -90 || point.Lat > 90 || point.Lon < -180 || point.Lon > 180 {
				return fmt.Errorf("activity detail %s has out-of-range route point", id)
			}
		}
	}
	return nil
}
